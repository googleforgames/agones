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

// Package main implements a client that reads a line
// from stdin and then sends it as a UDP packet to the
// -address flag
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

const (
	bufferSize = 1024
)

var (
	maxMessageSize = bufferSize - len([]byte("ACK: "))
)

// Program client reads a line in from stdin and sends it to a server as a UDP packet.
func main() {
	address := flag.String("address", "localhost:7654", "The address to send the UDP packets to")
	flag.Parse()

	log.Printf("Starting client, connecting to %s", *address)
	conn, err := net.Dial("udp", *address)
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close()

	go func() {
		b := make([]byte, bufferSize)
		for {
			n, err := conn.Read(b)
			if err != nil {
				log.Fatalf("Could not read packet: %v", err)
			}
			log.Printf("Received Packet: %s", b[:n])
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(">> Send Line: ")
	for scanner.Scan() {
		line := scanner.Bytes()
		len := len(line)

		if len > maxMessageSize {
			log.Printf("Messages cannot be larger than the buffer of %d bytes. Please try again.", maxMessageSize)
			continue
		}

		log.Printf("Sending %d bytes string %s via UDP", len, line)
		_, err = conn.Write(line)
		if err != nil {
			log.Fatalf("Could not write line %s to %s: %v", line, *address, err)
		}

		fmt.Print(">> Send Line: ")
	}
}
