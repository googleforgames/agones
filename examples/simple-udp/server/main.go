// Copyright 2017 Google Inc. All Rights Reserved.
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
	"flag"
	"log"
	"net"
	"os"
	"strings"

	"time"

	"agones.dev/agones/sdks/go"
)

// main starts a UDP server that received 1024 byte sized packets at at time
// converts the bytes to a string, and logs the output
func main() {
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
	defer conn.Close()

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	log.Print("Starting Health Ping")
	stop := make(chan bool)
	go doHealth(s, stop)

	log.Print("Marking this server as ready")
	// This tells Agones that the server is ready to receive connections.
	err = s.Ready()
	if err != nil {
		log.Fatalf("Could not send ready message")
	}

	b := make([]byte, 1024)
	for {
		n, sender, err := conn.ReadFrom(b)
		if err != nil {
			log.Fatalf("Could not read from udp stream: %v", err)
		}

		txt := strings.TrimSpace(string(b[:n]))
		log.Printf("Received packet from %v: %v", sender.String(), txt)
		switch txt {
		// shuts down the gameserver
		case "EXIT":
			log.Printf("Received EXIT command. Exiting.")
			// This tells Agones to shutdown this Game Server
			err := s.Shutdown()
			if err != nil {
				log.Printf("Could not shutdown")
			}
			os.Exit(0)
			break

		// turns off the health pings
		case "UNHEALTHY":
			close(stop)
			break
		}

		// echo it back
		ack := "ACK: " + txt + "\n"
		if _, err = conn.WriteTo([]byte(ack), sender); err != nil {
			log.Fatalf("Could not write to udp stream: %v", err)
		}
	}
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK, stop <-chan bool) {
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
