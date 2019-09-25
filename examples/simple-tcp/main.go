// Copyright 2019 Google LLC All Rights Reserved.
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

// Package main is a very simple echo TCP server
package main

import (
	"bufio"
	"flag"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
)

// main starts a TCP server that receives a message at a time
// (newline delineated), and echos the output.
func main() {
	go doSignal()

	port := flag.String("port", "7654", "The port to listen to tcp traffic on")
	flag.Parse()
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}

	log.Printf("Starting TCP server, listening on port %s", *port)
	ln, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Could not start tcp server: %v", err)
	}
	defer ln.Close() // nolint: errcheck

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	log.Print("Starting Health Ping")
	stop := make(chan struct{})
	go doHealth(s, stop)

	log.Print("Marking this server as ready")
	if err := s.Ready(); err != nil {
		log.Fatalf("Could not send ready message")
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Unable to accept incoming tcp connection: %v", err)
		}
		go handleConnection(conn, stop, s)
	}
}

// doSignal shutsdown on SIGTERM/SIGKILL
func doSignal() {
	stop := signals.NewStopChannel()
	<-stop
	log.Println("Exit signal received. Shutting down.")
	os.Exit(0)
}

// handleConnection services a single tcp connection to the server
func handleConnection(conn net.Conn, stop chan struct{}, s *sdk.SDK) {
	log.Printf("Client %s connected", conn.RemoteAddr().String())
	scanner := bufio.NewScanner(conn)
	for {
		if ok := scanner.Scan(); !ok {
			log.Printf("Client %s disconnected", conn.RemoteAddr().String())
			return
		}
		handleCommand(conn, scanner.Text(), stop, s)
	}
}

// respond responds to a given sender.
func respond(conn net.Conn, txt string) {
	log.Printf("Responding with %q", txt)
	if _, err := conn.Write([]byte(txt+"\n")); err != nil {
		log.Fatalf("Could not write to tcp stream: %v", err)
	}
}

func handleCommand(conn net.Conn, txt string, stop chan struct{}, s *sdk.SDK) {
	parts := strings.Split(strings.TrimSpace(txt), " ")

	log.Printf("parts: %v", parts)
	switch parts[0] {
	// shuts down the gameserver
	case "EXIT":
		respond(conn, "ACK: "+txt)
		exit(s)

	// turns off the health pings
	case "UNHEALTHY":
		close(stop)
	}

	respond(conn, "ACK: "+txt+"\n")
}

// exit shutdowns the server
func exit(s *sdk.SDK) {
	log.Printf("Received EXIT command. Exiting.")
	// This tells Agones to shutdown this Game Server
	if err := s.Shutdown(); err != nil {
		log.Printf("Could not call shutdown: %v", err)
	}
	os.Exit(0)
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK, stop <-chan struct{}) {
	tick := time.Tick(2 * time.Second)
	for {
		if err := sdk.Health(); err != nil {
			log.Fatalf("Could not send health ping: %v", err)
		}
		select {
		case <-stop:
			log.Print("Stopped health pings")
			return
		case <-tick:
		}
	}
}
