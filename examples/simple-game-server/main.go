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
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	coresdk "agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
)

// main starts a UDP or TCP server
func main() {
	go doSignal()

	port := flag.String("port", "7654", "The port to listen to traffic on")
	passthrough := flag.Bool("passthrough", false, "Get listening port from the SDK, rather than use the 'port' value")
	readyOnStart := flag.Bool("ready", true, "Mark this GameServer as Ready on startup")
	shutdownDelay := flag.Int("automaticShutdownDelayMin", 0, "If greater than zero, automatically shut down the server this many minutes after the server becomes allocated")
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

	if !*udp && !*tcp {
		log.Fatalf("No protocol enabled, exiting")
	}

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	log.Print("Starting Health Ping")
	stop := make(chan struct{})
	go doHealth(s, stop)

	if *passthrough {
		var gs *coresdk.GameServer
		gs, err = s.GameServer()
		if err != nil {
			log.Fatalf("Could not get gameserver port details: %s", err)
		}

		p := strconv.FormatInt(int64(gs.Status.Ports[0].Port), 10)
		port = &p
	}

	if *readyOnStart {
		log.Print("Marking this server as ready")
		ready(s)
	}

	if *shutdownDelay > 0 {
		shutdownAfterAllocation(s, *shutdownDelay)
	}

	if *tcp {
		go tcpListener(port, s, stop)
	}

	if *udp {
		go udpListener(port, s, stop)
	}

	// prevents program from quitting as the server is listening on goroutines
	for {}
}

// doSignal shutsdown on SIGTERM/SIGKILL
func doSignal() {
	stop := signals.NewStopChannel()
	<-stop
	log.Println("Exit signal received. Shutting down.")
	os.Exit(0)
}

// shutdownAfterAllocation creates a callback to automatically shut down
// the server a specified number of minutes after the server becomes
// allocated.
func shutdownAfterAllocation(s *sdk.SDK, shutdownDelay int) {
	err := s.WatchGameServer(func(gs *coresdk.GameServer) {
		if gs.Status.State == "Allocated" {
			time.Sleep(time.Duration(shutdownDelay) * time.Minute)
			shutdownErr := s.Shutdown()
			if shutdownErr != nil {
				log.Fatalf("Could not shutdown: %v", shutdownErr)
			}
		}
	})
	if err != nil {
		log.Fatalf("Could not watch Game Server events, %v", err)
	}
}

func handleResponse(txt string, s *sdk.SDK, stop chan struct{}) (response string, addACK bool, responseError error) {
	parts := strings.Split(strings.TrimSpace(txt), " ")
	response = txt
	addACK = true
	responseError = nil

	switch parts[0] {
	// shuts down the gameserver
	case "EXIT":
		// handle elsewhere, as we respond before exiting
		return

	// turns off the health pings
	case "UNHEALTHY":
		close(stop)

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
	}

	return
}

func udpListener(port *string, s *sdk.SDK, stop chan struct{}) {
	log.Printf("Starting UDP server, listening on port %s", *port)
	conn, err := net.ListenPacket("udp", ":"+*port)
	if err != nil {
		log.Fatalf("Could not start UDP server: %v", err)
	}
	defer conn.Close() // nolint: errcheck
	udpReadWriteLoop(conn, stop, s)
}

func udpReadWriteLoop(conn net.PacketConn, stop chan struct{}, s *sdk.SDK) {
	b := make([]byte, 1024)
	for {
		sender, txt := readPacket(conn, b)

		log.Printf("Received UDP: %v", txt)

		response, addACK, err := handleResponse(txt, s, stop)
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

func tcpListener(port *string, s *sdk.SDK, stop chan struct{}) {
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
		go tcpHandleConnection(conn, s, stop)
	}
}

// handleConnection services a single tcp connection to the server
func tcpHandleConnection(conn net.Conn, s *sdk.SDK, stop chan struct{}) {
	log.Printf("TCP Client %s connected", conn.RemoteAddr().String())
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		tcpHandleCommand(conn, scanner.Text(), s, stop)
	}
	log.Printf("TCP Client %s disconnected", conn.RemoteAddr().String())
}

func tcpHandleCommand(conn net.Conn, txt string, s *sdk.SDK, stop chan struct{}) {
	log.Printf("TCP txt: %v", txt)

	response, addACK, err := handleResponse(txt, s, stop)
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
	os.Exit(0)
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

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK, stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
		log.Printf("Health Ping")
		err := sdk.Health()
		if err != nil {
			log.Fatalf("Could not send health ping, %v", err)
		}
		select {
		case <-stop:
			log.Print("Stopped health pings")
			return
		case <-tick:
		}
	}
}
