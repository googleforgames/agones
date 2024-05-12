// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main is a very simple server with UDP (default), TCP, or both
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	coresdk "agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
)

// main starts a UDP or TCP server
func main() {
	sigCtx, _ := signals.NewSigKillContext()

	port := flag.String("port", "7654", "The port to listen to traffic on")
	passthrough := flag.Bool("passthrough", false, "Get listening port from the SDK, rather than use the 'port' value")
	readyOnStart := flag.Bool("ready", true, "Mark this GameServer as Ready on startup")
	shutdownDelayMin := flag.Int("automaticShutdownDelayMin", 0, "[Deprecated] If greater than zero, automatically shut down the server this many minutes after the server becomes allocated (please use automaticShutdownDelaySec instead)")
	shutdownDelaySec := flag.Int("automaticShutdownDelaySec", 0, "If greater than zero, automatically shut down the server this many seconds after the server becomes allocated (cannot be used if automaticShutdownDelayMin is set)")
	readyDelaySec := flag.Int("readyDelaySec", 0, "If greater than zero, wait this many seconds each time before marking the game server as ready")
	readyIterations := flag.Int("readyIterations", 0, "If greater than zero, return to a ready state this number of times before shutting down")
	gracefulTerminationDelaySec := flag.Int("gracefulTerminationDelaySec", 0, "Delay after we've been asked to terminate (by SIGKILL or automaticShutdownDelaySec)")
	udp := flag.Bool("udp", true, "Server will listen on UDP")
	tcp := flag.Bool("tcp", false, "Server will listen on TCP")

	flag.Parse()
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	if epass := os.Getenv("PASSTHROUGH"); epass != "" {
		p := strings.ToUpper(epass) == "TRUE"
		passthrough = &p
	}
	if eready := os.Getenv("READY"); eready != "" {
		r := strings.ToUpper(eready) == "TRUE"
		readyOnStart = &r
	}
	if eudp := os.Getenv("UDP"); eudp != "" {
		u := strings.ToUpper(eudp) == "TRUE"
		udp = &u
	}
	if etcp := os.Getenv("TCP"); etcp != "" {
		t := strings.ToUpper(etcp) == "TRUE"
		tcp = &t
	}

	// Check for incompatible flags.
	if *shutdownDelayMin > 0 && *shutdownDelaySec > 0 {
		log.Fatalf("Cannot set both --automaticShutdownDelayMin and --automaticShutdownDelaySec")
	}
	if *readyIterations > 0 && *shutdownDelayMin <= 0 && *shutdownDelaySec <= 0 {
		log.Fatalf("Must set a shutdown delay if using ready iterations")
	}

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	log.Print("Starting Health Ping")
	ctx, cancel := context.WithCancel(context.Background())
	go doHealth(s, ctx)

	if *passthrough {
		var gs *coresdk.GameServer
		gs, err = s.GameServer()
		if err != nil {
			log.Fatalf("Could not get gameserver port details: %s", err)
		}

		p := strconv.FormatInt(int64(gs.Status.Ports[0].Port), 10)
		port = &p
	}

	if *tcp {
		go tcpListener(port, s, cancel)
	}

	if *udp {
		go udpListener(port, s, cancel)
	}

	if *shutdownDelaySec > 0 {
		shutdownAfterNAllocations(s, *readyIterations, *shutdownDelaySec)
	} else if *shutdownDelayMin > 0 {
		shutdownAfterNAllocations(s, *readyIterations, *shutdownDelayMin*60)
	}

	if *readyOnStart {
		if *readyDelaySec > 0 {
			log.Printf("Waiting %d seconds before moving to ready", *readyDelaySec)
			time.Sleep(time.Duration(*readyDelaySec) * time.Second)
		}
		log.Print("Marking this server as ready")
		ready(s)
	}

	<-sigCtx.Done()
	log.Printf("Waiting %d seconds before exiting", *gracefulTerminationDelaySec)
	time.Sleep(time.Duration(*gracefulTerminationDelaySec) * time.Second)
	os.Exit(0)
}

// shutdownAfterNAllocations creates a callback to automatically shut down
// the server a specified number of seconds after the server becomes
// allocated the Nth time.
//
// The algorithm is:
//
//  1. Move the game server back to ready N times after it is allocated
//  2. Shutdown the game server after the Nth time is becomes allocated
//
// This follows the integration pattern documented on the website at
// https://agones.dev/site/docs/integration-patterns/reusing-gameservers/
func shutdownAfterNAllocations(s *sdk.SDK, readyIterations, shutdownDelaySec int) {
	gs, err := s.GameServer()
	if err != nil {
		log.Fatalf("Could not get game server: %v", err)
	}
	log.Printf("Initial game Server state = %s", gs.Status.State)

	m := sync.Mutex{} // protects the following two variables
	lastAllocated := gs.ObjectMeta.Annotations["agones.dev/last-allocated"]
	remainingIterations := readyIterations

	if err := s.WatchGameServer(func(gs *coresdk.GameServer) {
		m.Lock()
		defer m.Unlock()
		la := gs.ObjectMeta.Annotations["agones.dev/last-allocated"]
		log.Printf("Watch Game Server callback fired. State = %s, Last Allocated = %q", gs.Status.State, la)
		if lastAllocated != la {
			log.Println("Game Server Allocated")
			lastAllocated = la
			remainingIterations--
			// Run asynchronously
			go func(iterations int) {
				time.Sleep(time.Duration(shutdownDelaySec) * time.Second)

				if iterations > 0 {
					log.Println("Moving Game Server back to Ready")
					readyErr := s.Ready()
					if readyErr != nil {
						log.Fatalf("Could not set game server to ready: %v", readyErr)
					}
					log.Println("Game Server is Ready")
					return
				}

				log.Println("Moving Game Server to Shutdown")
				if shutdownErr := s.Shutdown(); shutdownErr != nil {
					log.Fatalf("Could not shutdown game server: %v", shutdownErr)
				}
				// The process will exit when Agones removes the pod and the
				// container receives the SIGTERM signal
				return
			}(remainingIterations)
		}
	}); err != nil {
		log.Fatalf("Could not watch Game Server events, %v", err)
	}
}

func handleResponse(txt string, s *sdk.SDK, cancel context.CancelFunc) (response string, addACK bool, responseError error) {
	parts := strings.Split(strings.TrimSpace(txt), " ")
	response = txt
	addACK = true
	responseError = nil
	var err error

	switch parts[0] {
	// shuts down the gameserver
	case "EXIT":
		// handle elsewhere, as we respond before exiting
		return

	// turns off the health pings
	case "UNHEALTHY":
		cancel()

	case "GAMESERVER":
		response = gameServerName(s)
		addACK = false

	case "READY":
		ready(s)

	case "ALLOCATE":
		allocate(s)

	case "RESERVE":
		if len(parts) != 2 {
			response = "Invalid RESERVE, should have 1 argument"
			responseError = fmt.Errorf("Invalid RESERVE, should have 1 argument")
		}
		if dur, err := time.ParseDuration(parts[1]); err != nil {
			response = fmt.Sprintf("%s\n", err)
			responseError = err
		} else {
			reserve(s, dur)
		}

	case "WATCH":
		watchGameServerEvents(s)

	case "LABEL":
		switch len(parts) {
		case 1:
			// legacy format
			setLabel(s, "timestamp", strconv.FormatInt(time.Now().Unix(), 10))
		case 3:
			setLabel(s, parts[1], parts[2])
		default:
			response = "Invalid LABEL command, must use zero or 2 arguments"
			responseError = fmt.Errorf("Invalid LABEL command, must use zero or 2 arguments")
		}

	case "CRASH":
		log.Print("Crashing.")
		os.Exit(1)
		return "", false, nil

	case "ANNOTATION":
		switch len(parts) {
		case 1:
			// legacy format
			setAnnotation(s, "timestamp", time.Now().UTC().String())
		case 3:
			setAnnotation(s, parts[1], parts[2])
		default:
			response = "Invalid ANNOTATION command, must use zero or 2 arguments"
			responseError = fmt.Errorf("Invalid ANNOTATION command, must use zero or 2 arguments")
		}

	case "PLAYER_CAPACITY":
		switch len(parts) {
		case 1:
			response = getPlayerCapacity(s)
			addACK = false
		case 2:
			if cap, err := strconv.Atoi(parts[1]); err != nil {
				response = fmt.Sprintf("%s", err)
				responseError = err
			} else {
				setPlayerCapacity(s, int64(cap))
			}
		default:
			response = "Invalid PLAYER_CAPACITY, should have 0 or 1 arguments"
			responseError = fmt.Errorf("Invalid PLAYER_CAPACITY, should have 0 or 1 arguments")
		}

	case "PLAYER_CONNECT":
		if len(parts) < 2 {
			response = "Invalid PLAYER_CONNECT, should have 1 arguments"
			responseError = fmt.Errorf("Invalid PLAYER_CONNECT, should have 1 arguments")
			return
		}
		playerConnect(s, parts[1])

	case "PLAYER_DISCONNECT":
		if len(parts) < 2 {
			response = "Invalid PLAYER_DISCONNECT, should have 1 arguments"
			responseError = fmt.Errorf("Invalid PLAYER_DISCONNECT, should have 1 arguments")
			return
		}
		playerDisconnect(s, parts[1])

	case "PLAYER_CONNECTED":
		if len(parts) < 2 {
			response = "Invalid PLAYER_CONNECTED, should have 1 arguments"
			responseError = fmt.Errorf("Invalid PLAYER_CONNECTED, should have 1 arguments")
			return
		}
		response = playerIsConnected(s, parts[1])
		addACK = false

	case "GET_PLAYERS":
		response = getConnectedPlayers(s)
		addACK = false

	case "PLAYER_COUNT":
		response = getPlayerCount(s)
		addACK = false

	case "GET_COUNTER_COUNT":
		if len(parts) < 2 {
			response = "Invalid GET_COUNTER_COUNT, should have 1 arguments"
			responseError = fmt.Errorf("Invalid GET_COUNTER_COUNT, should have 1 arguments")
			return
		}
		response, responseError = getCounterCount(s, parts[1])
		addACK = false

	case "INCREMENT_COUNTER":
		if len(parts) < 3 {
			response = "Invalid INCREMENT_COUNTER, should have 2 arguments"
			responseError = fmt.Errorf("Invalid INCREMENT_COUNTER, should have 2 arguments")
			return
		}
		response, err = incrementCounter(s, parts[1], parts[2])
		addACK = false

	case "DECREMENT_COUNTER":
		if len(parts) < 3 {
			response = "Invalid DECREMENT_COUNTER, should have 2 arguments"
			responseError = fmt.Errorf("Invalid DECREMENT_COUNTER, should have 2 arguments")
			return
		}
		response, err = decrementCounter(s, parts[1], parts[2])
		addACK = false

	case "SET_COUNTER_COUNT":
		if len(parts) < 3 {
			response = "Invalid SET_COUNTER_COUNT, should have 2 arguments"
			responseError = fmt.Errorf("Invalid SET_COUNTER_COUNT, should have 2 arguments")
			return
		}
		response, err = setCounterCount(s, parts[1], parts[2])
		addACK = false

	case "GET_COUNTER_CAPACITY":
		if len(parts) < 2 {
			response = "Invalid GET_COUNTER_CAPACITY, should have 1 arguments"
			responseError = fmt.Errorf("Invalid GET_COUNTER_CAPACITY, should have 1 arguments")
			return
		}
		response, responseError = getCounterCapacity(s, parts[1])
		addACK = false

	case "SET_COUNTER_CAPACITY":
		if len(parts) < 3 {
			response = "Invalid SET_COUNTER_CAPACITY, should have 2 arguments"
			responseError = fmt.Errorf("Invalid SET_COUNTER_CAPACITY, should have 2 arguments")
			return
		}
		response, err = setCounterCapacity(s, parts[1], parts[2])
		addACK = false

	case "GET_LIST_CAPACITY":
		if len(parts) < 2 {
			response = "Invalid GET_LIST_CAPACITY, should have 1 arguments"
			responseError = fmt.Errorf("Invalid GET_LIST_CAPACITY, should have 1 arguments")
			return
		}
		response, responseError = getListCapacity(s, parts[1])
		addACK = false

	case "SET_LIST_CAPACITY":
		if len(parts) < 3 {
			response = "Invalid SET_LIST_CAPACITY, should have 2 arguments"
			responseError = fmt.Errorf("Invalid SET_LIST_CAPACITY, should have 2 arguments")
			return
		}
		response, err = setListCapacity(s, parts[1], parts[2])
		addACK = false

	case "LIST_CONTAINS":
		if len(parts) < 3 {
			response = "Invalid LIST_CONTAINS, should have 2 arguments"
			responseError = fmt.Errorf("Invalid LIST_CONTAINS, should have 2 arguments")
			return
		}
		response, responseError = listContains(s, parts[1], parts[2])
		addACK = false

	case "GET_LIST_LENGTH":
		if len(parts) < 2 {
			response = "Invalid GET_LIST_LENGTH, should have 1 arguments"
			responseError = fmt.Errorf("Invalid GET_LIST_LENGTH, should have 1 arguments")
			return
		}
		response, responseError = getListLength(s, parts[1])
		addACK = false

	case "GET_LIST_VALUES":
		if len(parts) < 2 {
			response = "Invalid GET_LIST_VALUES, should have 1 arguments"
			responseError = fmt.Errorf("Invalid GET_LIST_VALUES, should have 1 arguments")
			return
		}
		response, responseError = getListValues(s, parts[1])
		addACK = false

	case "APPEND_LIST_VALUE":
		if len(parts) < 3 {
			response = "Invalid APPEND_LIST_VALUE, should have 2 arguments"
			responseError = fmt.Errorf("Invalid APPEND_LIST_VALUE, should have 2 arguments")
			return
		}
		response, err = appendListValue(s, parts[1], parts[2])
		addACK = false

	case "DELETE_LIST_VALUE":
		if len(parts) < 3 {
			response = "Invalid DELETE_LIST_VALUE, should have 2 arguments"
			responseError = fmt.Errorf("Invalid DELETE_LIST_VALUE, should have 2 arguments")
			return
		}
		response, err = deleteListValue(s, parts[1], parts[2])
		addACK = false
	}

	if err != nil {
		return err.Error(), addACK, err
	}

	return
}

func udpListener(port *string, s *sdk.SDK, cancel context.CancelFunc) {
	log.Printf("Starting UDP server, listening on port %s", *port)
	conn, err := net.ListenPacket("udp", ":"+*port)
	if err != nil {
		log.Fatalf("Could not start UDP server: %v", err)
	}
	defer conn.Close() // nolint: errcheck
	udpReadWriteLoop(conn, cancel, s)
}

func udpReadWriteLoop(conn net.PacketConn, cancel context.CancelFunc, s *sdk.SDK) {
	b := make([]byte, 1024)
	for {
		sender, txt := readPacket(conn, b)

		log.Printf("Received UDP: %v", txt)

		response, addACK, err := handleResponse(txt, s, cancel)
		if err != nil {
			response = "ERROR: " + response + "\n"
		} else if addACK {
			response = "ACK: " + response + "\n"
		}

		udpRespond(conn, sender, response)

		if txt == "EXIT" {
			exit(s)
		}
	}
}

// respond responds to a given sender.
func udpRespond(conn net.PacketConn, sender net.Addr, txt string) {
	if _, err := conn.WriteTo([]byte(txt), sender); err != nil {
		log.Fatalf("Could not write to udp stream: %v", err)
	}
}

func tcpListener(port *string, s *sdk.SDK, cancel context.CancelFunc) {
	log.Printf("Starting TCP server, listening on port %s", *port)
	ln, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Could not start TCP server: %v", err)
	}
	defer ln.Close() // nolint: errcheck

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Unable to accept incoming TCP connection: %v", err)
		}
		go tcpHandleConnection(conn, s, cancel)
	}
}

// handleConnection services a single tcp connection to the server
func tcpHandleConnection(conn net.Conn, s *sdk.SDK, cancel context.CancelFunc) {
	log.Printf("TCP Client %s connected", conn.RemoteAddr().String())
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		tcpHandleCommand(conn, scanner.Text(), s, cancel)
	}
	log.Printf("TCP Client %s disconnected", conn.RemoteAddr().String())
}

func tcpHandleCommand(conn net.Conn, txt string, s *sdk.SDK, cancel context.CancelFunc) {
	log.Printf("TCP txt: %v", txt)

	response, addACK, err := handleResponse(txt, s, cancel)
	if err != nil {
		response = "ERROR: " + response + "\n"
	} else if addACK {
		response = "ACK TCP: " + response + "\n"
	}

	tcpRespond(conn, response)

	if response == "EXIT" {
		exit(s)
	}
}

// respond responds to a given sender.
func tcpRespond(conn net.Conn, txt string) {
	log.Printf("Responding to TCP with %q", txt)
	if _, err := conn.Write([]byte(txt + "\n")); err != nil {
		log.Fatalf("Could not write to TCP stream: %v", err)
	}
}

// ready attempts to mark this gameserver as ready
func ready(s *sdk.SDK) {
	err := s.Ready()
	if err != nil {
		log.Fatalf("Could not send ready message")
	}
}

// allocate attempts to allocate this gameserver
func allocate(s *sdk.SDK) {
	err := s.Allocate()
	if err != nil {
		log.Fatalf("could not allocate gameserver: %v", err)
	}
}

// reserve for 10 seconds
func reserve(s *sdk.SDK, duration time.Duration) {
	if err := s.Reserve(duration); err != nil {
		log.Fatalf("could not reserve gameserver: %v", err)
	}
}

// readPacket reads a string from the connection
func readPacket(conn net.PacketConn, b []byte) (net.Addr, string) {
	n, sender, err := conn.ReadFrom(b)
	if err != nil {
		log.Fatalf("Could not read from udp stream: %v", err)
	}
	txt := strings.TrimSpace(string(b[:n]))
	log.Printf("Received packet from %v: %v", sender.String(), txt)
	return sender, txt
}

// exit shutdowns the server
func exit(s *sdk.SDK) {
	log.Printf("Received EXIT command. Exiting.")
	// This tells Agones to shutdown this Game Server
	shutdownErr := s.Shutdown()
	if shutdownErr != nil {
		log.Printf("Could not shutdown")
	}
	// The process will exit when Agones removes the pod and the
	// container receives the SIGTERM signal
}

// gameServerName returns the GameServer name
func gameServerName(s *sdk.SDK) string {
	var gs *coresdk.GameServer
	gs, err := s.GameServer()
	if err != nil {
		log.Fatalf("Could not retrieve GameServer: %v", err)
	}
	var j []byte
	j, err = json.Marshal(gs)
	if err != nil {
		log.Fatalf("error mashalling GameServer to JSON: %v", err)
	}
	log.Printf("GameServer: %s \n", string(j))
	return "NAME: " + gs.ObjectMeta.Name + "\n"
}

// watchGameServerEvents creates a callback to log when
// gameserver events occur
func watchGameServerEvents(s *sdk.SDK) {
	err := s.WatchGameServer(func(gs *coresdk.GameServer) {
		j, err := json.Marshal(gs)
		if err != nil {
			log.Fatalf("error mashalling GameServer to JSON: %v", err)
		}
		log.Printf("GameServer Event: %s \n", string(j))
	})
	if err != nil {
		log.Fatalf("Could not watch Game Server events, %v", err)
	}
}

// setAnnotation sets a given annotation
func setAnnotation(s *sdk.SDK, key, value string) {
	log.Printf("Setting annotation %v=%v", key, value)
	err := s.SetAnnotation(key, value)
	if err != nil {
		log.Fatalf("could not set annotation: %v", err)
	}
}

// setLabel sets a given label
func setLabel(s *sdk.SDK, key, value string) {
	log.Printf("Setting label %v=%v", key, value)
	// label values can only be alpha, - and .
	err := s.SetLabel(key, value)
	if err != nil {
		log.Fatalf("could not set label: %v", err)
	}
}

// setPlayerCapacity sets the player capacity to the given value
func setPlayerCapacity(s *sdk.SDK, capacity int64) {
	log.Printf("Setting Player Capacity to %d", capacity)
	if err := s.Alpha().SetPlayerCapacity(capacity); err != nil {
		log.Fatalf("could not set capacity: %v", err)
	}
}

// getPlayerCapacity returns the current player capacity as a string
func getPlayerCapacity(s *sdk.SDK) string {
	log.Print("Getting Player Capacity")
	capacity, err := s.Alpha().GetPlayerCapacity()
	if err != nil {
		log.Fatalf("could not get capacity: %v", err)
	}
	return strconv.FormatInt(capacity, 10) + "\n"
}

// playerConnect connects a given player
func playerConnect(s *sdk.SDK, id string) {
	log.Printf("Connecting Player: %s", id)
	if _, err := s.Alpha().PlayerConnect(id); err != nil {
		log.Fatalf("could not connect player: %v", err)
	}
}

// playerDisconnect disconnects a given player
func playerDisconnect(s *sdk.SDK, id string) {
	log.Printf("Disconnecting Player: %s", id)
	if _, err := s.Alpha().PlayerDisconnect(id); err != nil {
		log.Fatalf("could not disconnect player: %v", err)
	}
}

// playerIsConnected returns a bool as a string if a player is connected
func playerIsConnected(s *sdk.SDK, id string) string {
	log.Printf("Checking if player %s is connected", id)

	connected, err := s.Alpha().IsPlayerConnected(id)
	if err != nil {
		log.Fatalf("could not retrieve if player is connected: %v", err)
	}

	return strconv.FormatBool(connected) + "\n"
}

// getConnectedPlayers returns a comma delimeted list of connected players
func getConnectedPlayers(s *sdk.SDK) string {
	log.Print("Retrieving connected player list")
	list, err := s.Alpha().GetConnectedPlayers()
	if err != nil {
		log.Fatalf("could not retrieve connected players: %s", err)
	}

	return strings.Join(list, ",") + "\n"
}

// getPlayerCount returns the count of connected players as a string
func getPlayerCount(s *sdk.SDK) string {
	log.Print("Retrieving connected player count")
	count, err := s.Alpha().GetPlayerCount()
	if err != nil {
		log.Fatalf("could not retrieve player count: %s", err)
	}
	return strconv.FormatInt(count, 10) + "\n"
}

// getCounterCount returns the Count of the given Counter as a string
func getCounterCount(s *sdk.SDK, counterName string) (string, error) {
	log.Printf("Retrieving Counter %s Count", counterName)
	count, err := s.Beta().GetCounterCount(counterName)
	if err != nil {
		log.Printf("Error getting Counter %s Count: %s", counterName, err)
		return strconv.FormatInt(count, 10), err
	}
	return "COUNTER: " + strconv.FormatInt(count, 10) + "\n", nil
}

// incrementCounter returns the if the Counter Count was incremented successfully or not
func incrementCounter(s *sdk.SDK, counterName string, amount string) (string, error) {
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("Could not increment Counter %s by unparseable amount %s: %s", counterName, amount, err)
	}
	log.Printf("Incrementing Counter %s Count by amount %d", counterName, amountInt)
	err = s.Beta().IncrementCounter(counterName, amountInt)
	if err != nil {
		log.Printf("Error incrementing Counter %s Count by amount %d: %s", counterName, amountInt, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// decrementCounter returns if the Counter Count was decremented successfully or not
func decrementCounter(s *sdk.SDK, counterName string, amount string) (string, error) {
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("could not decrement Counter %s by unparseable amount %s: %s", counterName, amount, err)
	}
	log.Printf("Decrementing Counter %s Count by amount %d", counterName, amountInt)
	err = s.Beta().DecrementCounter(counterName, amountInt)
	if err != nil {
		log.Printf("Error decrementing Counter %s Count by amount %d: %s", counterName, amountInt, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// setCounterCount returns the if the Counter was set to a new Count successfully or not
func setCounterCount(s *sdk.SDK, counterName string, amount string) (string, error) {
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("could not set Counter %s to unparseable amount %s: %s", counterName, amount, err)
	}
	log.Printf("Setting Counter %s Count to amount %d", counterName, amountInt)
	err = s.Beta().SetCounterCount(counterName, amountInt)
	if err != nil {
		log.Printf("Error setting Counter %s Count by amount %d: %s", counterName, amountInt, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// getCounterCapacity returns the Capacity of the given Counter as a string
func getCounterCapacity(s *sdk.SDK, counterName string) (string, error) {
	log.Printf("Retrieving Counter %s Capacity", counterName)
	count, err := s.Beta().GetCounterCapacity(counterName)
	if err != nil {
		log.Printf("Error getting Counter %s Capacity: %s", counterName, err)
		return strconv.FormatInt(count, 10), err
	}
	return "CAPACITY: " + strconv.FormatInt(count, 10) + "\n", nil
}

// setCounterCapacity returns the if the Counter was set to a new Capacity successfully or not
func setCounterCapacity(s *sdk.SDK, counterName string, amount string) (string, error) {
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("could not set Counter %s to unparseable amount %s: %s", counterName, amount, err)
	}
	log.Printf("Setting Counter %s Capacity to amount %d", counterName, amountInt)
	err = s.Beta().SetCounterCapacity(counterName, amountInt)
	if err != nil {
		log.Printf("Error setting Counter %s Capacity to amount %d: %s", counterName, amountInt, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// getListCapacity returns the Capacity of the given List as a string
func getListCapacity(s *sdk.SDK, listName string) (string, error) {
	log.Printf("Retrieving List %s Capacity", listName)
	capacity, err := s.Beta().GetListCapacity(listName)
	if err != nil {
		log.Printf("Error getting List %s Capacity: %s", listName, err)
		return strconv.FormatInt(capacity, 10), err
	}
	return "CAPACITY: " + strconv.FormatInt(capacity, 10) + "\n", nil
}

// setListCapacity returns if the List was set to a new Capacity successfully or not
func setListCapacity(s *sdk.SDK, listName string, amount string) (string, error) {
	amountInt, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		return "", fmt.Errorf("could not set List %s to unparseable amount %s: %s", listName, amount, err)
	}
	log.Printf("Setting List %s Capacity to amount %d", listName, amountInt)
	err = s.Beta().SetListCapacity(listName, amountInt)
	if err != nil {
		log.Printf("Error setting List %s Capacity to amount %d: %s", listName, amountInt, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// listContains returns true if the given value is in the given List, false otherwise
func listContains(s *sdk.SDK, listName string, value string) (string, error) {
	log.Printf("Getting List %s contains value %s", listName, value)
	ok, err := s.Beta().ListContains(listName, value)
	if err != nil {
		log.Printf("Error getting List %s contains value %s: %s", listName, value, err)
		return strconv.FormatBool(ok), err
	}
	return "FOUND: " + strconv.FormatBool(ok) + "\n", nil
}

// getListLength returns the length (number of values) of the given List as a string
func getListLength(s *sdk.SDK, listName string) (string, error) {
	log.Printf("Getting List %s length", listName)
	length, err := s.Beta().GetListLength(listName)
	if err != nil {
		log.Printf("Error getting List %s length: %s", listName, err)
		return strconv.Itoa(length), err
	}
	return "LENGTH: " + strconv.Itoa(length) + "\n", nil
}

// getListValues return the values in the given List as a comma delineated string
func getListValues(s *sdk.SDK, listName string) (string, error) {
	log.Printf("Getting List %s values", listName)
	values, err := s.Beta().GetListValues(listName)
	if err != nil {
		log.Printf("Error getting List %s values: %s", listName, err)
		return "INVALID LIST NAME", err
	}
	if len(values) > 0 {
		return "VALUES: " + strings.Join(values, ",") + "\n", nil
	}
	return "VALUES: <none>\n", nil
}

// appendListValue returns if the given value was successfuly added to the List or not
func appendListValue(s *sdk.SDK, listName string, value string) (string, error) {
	log.Printf("Appending Value %s to List %s", value, listName)
	err := s.Beta().AppendListValue(listName, value)
	if err != nil {
		log.Printf("Error appending Value %s to List %s: %s", value, listName, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// deleteListValue returns if the given value was successfuly deleted from the List or not
func deleteListValue(s *sdk.SDK, listName string, value string) (string, error) {
	log.Printf("Deleting Value %s from List %s", value, listName)
	err := s.Beta().DeleteListValue(listName, value)
	if err != nil {
		log.Printf("Error deleting Value %s to List %s: %s", value, listName, err)
		return "", err
	}
	return "SUCCESS\n", nil
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK, ctx context.Context) {
	tick := time.Tick(2 * time.Second)
	for {
		log.Printf("Health Ping")
		err := sdk.Health()
		if err != nil {
			log.Fatalf("Could not send health ping, %v", err)
		}
		select {
		case <-ctx.Done():
			log.Print("Stopped health pings")
			return
		case <-tick:
		}
	}
}
