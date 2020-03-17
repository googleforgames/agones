// Copyright 2018 Google LLC All Rights Reserved.
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

// Package main is a very simple echo UDP server
package main

import (
	"encoding/json"
	"flag"
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

// main starts a UDP server that received 1024 byte sized packets at at time
// converts the bytes to a string, and logs the output
func main() {
	go doSignal()

	port := flag.String("port", "7654", "The port to listen to udp traffic on")
	passthrough := flag.Bool("passthrough", false, "Get listening port from the SDK, rather than use the 'port' value")
	readyOnStart := flag.Bool("ready", true, "Mark this GameServer as Ready on startup")
	shutdownDelay := flag.Int("automaticShutdownDelayMin", 0, "If greater than zero, automatically shut down the server this many minutes after the server becomes allocated")
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

	log.Printf("Starting UDP server, listening on port %s", *port)
	conn, err := net.ListenPacket("udp", ":"+*port)
	if err != nil {
		log.Fatalf("Could not start udp server: %v", err)
	}
	defer conn.Close() // nolint: errcheck

	if *readyOnStart {
		log.Print("Marking this server as ready")
		ready(s)
	}

	if *shutdownDelay > 0 {
		shutdownAfterAllocation(s, *shutdownDelay)
	}
	readWriteLoop(conn, stop, s)
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
			go func() {
				time.Sleep(time.Duration(shutdownDelay) * time.Minute)
				shutdownErr := s.Shutdown()
				for shutdownErr != nil {
					log.Printf("Could not shutdown: %v", shutdownErr)
					// TODO: Add exponential backoff
					time.Sleep(10 * time.Second)
					shutdownErr = s.Shutdown()
				}
			}()
		}
	})
	if err != nil {
		log.Fatalf("Could not watch Game Server events, %v", err)
	}
}

func readWriteLoop(conn net.PacketConn, stop chan struct{}, s *sdk.SDK) {
	b := make([]byte, 1024)
	for {
		sender, txt := readPacket(conn, b)
		parts := strings.Split(strings.TrimSpace(txt), " ")

		switch parts[0] {
		// shuts down the gameserver
		case "EXIT":
			// respond here, as we os.Exit() before we get to below
			respond(conn, sender, "ACK: "+txt+"\n")
			exit(s)

		// turns off the health pings
		case "UNHEALTHY":
			close(stop)

		case "GAMESERVER":
			respond(conn, sender, gameServerName(s))

		case "READY":
			ready(s)

		case "ALLOCATE":
			allocate(s)

		case "RESERVE":
			reserve(s)

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
				respond(conn, sender, "ERROR: Invalid LABEL command, must use zero or 2 arguments")
				continue
			}

		case "CRASH":
			log.Print("Crashing.")
			os.Exit(1)

		case "ANNOTATION":
			switch len(parts) {
			case 1:
				// legacy format
				setAnnotation(s, "timestamp", time.Now().UTC().String())
			case 3:
				setAnnotation(s, parts[1], parts[2])
			default:
				respond(conn, sender, "ERROR: Invalid ANNOTATION command, must use zero or 2 arguments\n")
				continue
			}
		}

		respond(conn, sender, "ACK: "+txt+"\n")
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
func reserve(s *sdk.SDK) {
	if err := s.Reserve(10 * time.Second); err != nil {
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

// respond responds to a given sender.
func respond(conn net.PacketConn, sender net.Addr, txt string) {
	if _, err := conn.WriteTo([]byte(txt), sender); err != nil {
		log.Fatalf("Could not write to udp stream: %v", err)
	}
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

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK, stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
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
