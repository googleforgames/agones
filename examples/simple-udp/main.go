// Copyright 2018 Google Inc. All Rights Reserved.
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
	"agones.dev/agones/sdks/go"
)

// main starts a UDP server that received 1024 byte sized packets at at time
// converts the bytes to a string, and logs the output
func main() {
	go doSignal()

	port := flag.String("port", "7654", "The port to listen to udp traffic on")
	flag.Parse()
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}

	log.Printf("Starting UDP server, listening on port %s", *port)
	conn, err := net.ListenPacket("udp", ":"+*port)
	if err != nil {
		log.Fatalf("Could not start udp server: %v", err)
	}
	defer conn.Close() // nolint: errcheck

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	log.Print("Starting Health Ping")
	stop := make(chan struct{})
	go doHealth(s, stop)

	log.Print("Marking this server as ready")
	// This tells Agones that the server is ready to receive connections.
	err = s.Ready()
	if err != nil {
		log.Fatalf("Could not send ready message")
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

func readWriteLoop(conn net.PacketConn, stop chan struct{}, s *sdk.SDK) {
	b := make([]byte, 1024)
	for {
		sender, txt := readPacket(conn, b)
		switch txt {

		// shuts down the gameserver
		case "EXIT":
			exit(s)

		// turns off the health pings
		case "UNHEALTHY":
			close(stop)

		case "GAMESERVER":
			writeGameServerName(s, conn, sender)

		case "WATCH":
			watchGameServerEvents(s)

		case "LABEL":
			setLabel(s)

		case "ANNOTATION":
			setAnnotation(s)
		}

		ack(conn, sender, txt)
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

// ack echoes it back, with an ACK
func ack(conn net.PacketConn, sender net.Addr, txt string) {
	ack := "ACK: " + txt + "\n"
	if _, err := conn.WriteTo([]byte(ack), sender); err != nil {
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

// writes the GameServer name to the connection UDP stream
func writeGameServerName(s *sdk.SDK, conn net.PacketConn, sender net.Addr) {
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
	msg := "NAME: " + gs.ObjectMeta.Name + "\n"
	if _, err = conn.WriteTo([]byte(msg), sender); err != nil {
		log.Fatalf("Could not write to udp stream: %v", err)
	}
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
func setAnnotation(s *sdk.SDK) {
	log.Print("Setting annotation")
	err := s.SetAnnotation("timestamp", time.Now().UTC().String())
	if err != nil {
		log.Fatalf("could not set annotation: %v", err)
	}
}

// setLabel sets a given label
func setLabel(s *sdk.SDK) {
	log.Print("Setting label")
	// label values can only be alpha, - and .
	err := s.SetLabel("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
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
