// Copyright 2024 Google LLC All Rights Reserved.
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

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
)

// main starts a UDP or TCP server
func main() {
	sigCtx, _ := signals.NewSigKillContext()

	port := flag.String("port", "7654", "The port to listen to traffic on")
	udp := flag.Bool("udp", true, "Server will listen on UDP")
	tcp := flag.Bool("tcp", false, "Server will listen on TCP")
	endpoint := flag.String("endpoint", "", "The full base URL to send API requests to")

	flag.Parse()
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	if eudp := os.Getenv("UDP"); eudp != "" {
		u := strings.ToUpper(eudp) == "TRUE"
		udp = &u
	}
	if etcp := os.Getenv("TCP"); etcp != "" {
		t := strings.ToUpper(etcp) == "TRUE"
		tcp = &t
	}
	if endp := os.Getenv("ENDPOINT"); endp != "" {
		endpoint = &endp
	}

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}


	log.Print("Starting Health Ping")
	// TODO: Is "cancel" still being used even if no method explicitly uses it?
	ctx, cancel := context.WithCancel(context.Background())
	go doHealth(s, ctx)

	log.Printf("Creating Client at endpoint %s", *endpoint)
	clientConn, err := initClient(*endpoint)

	if *udp {
		go udpListener(port, s, cancel, clientConn)
	}

	if *tcp {
		go tcpListener(port, s, cancel, clientConn)
	}

	log.Print("Marking this server as ready")
	ready(s)

	<-sigCtx.Done()
	os.Exit(0)
}

func initClient(endpoint string) (*connection, error) {
	// TODO: create option for a client certificate
	client := &http.Client{}
	conn := &connection{client: client, endpoint: endpoint}
	return conn, nil
}

type connection struct {
	client   *http.Client
	endpoint string // full base URL for API requests
	// TODO: create options for routes off the base URL
}

type GenAIRequest struct {
	MaxOutputTokens int     `json:"max_output_tokens"`
	Prompt          string  `json:"prompt"`
	Temperature     float64 `json:"temperature"`
	TopK            int     `json:"top_k"`
	TopP            float64 `json:"top_p"`
}

func handleGenAIRequest(txt string, clientConn *connection) (string, error) {
	jsonRequest := GenAIRequest{
		MaxOutputTokens: 256,
		Prompt: txt,
		Temperature: 0.2,
		TopK: 40,
		TopP: 0.8,
	}
	jsonStr, err := json.Marshal(jsonRequest)
	if err != nil {
		return "unable to marshal json request", err
	}

	req, err := http.NewRequest("POST", clientConn.endpoint, bytes.NewBuffer(jsonStr))
	if err != nil {
		return "unable create http POST request", err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := clientConn.client.Do(req)
	if err != nil {
		return "Post request error", err
	}

	responseBody, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	body := string(responseBody)

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Status: %s, Body: %s", resp.Status, body)
	}
	return string(responseBody) + "\n", err
}

func udpListener(port *string, s *sdk.SDK, cancel context.CancelFunc, clientConn *connection) {
	log.Printf("Starting UDP server, listening on port %s", *port)
	conn, err := net.ListenPacket("udp", ":"+*port)
	if err != nil {
		log.Fatalf("Could not start UDP server: %v", err)
	}
	defer conn.Close() // nolint: errcheck
	udpReadWriteLoop(conn, cancel, s, clientConn)
}

func udpReadWriteLoop(conn net.PacketConn, cancel context.CancelFunc, s *sdk.SDK, clientConn *connection) {
	b := make([]byte, 1024)
	for {
		sender, txt := readPacket(conn, b)

		log.Printf("Received UDP: %v", txt)

		if txt == "EXIT" {
			exit(s)
		}

		// TODO: handle in go routine
		response, err := handleGenAIRequest(txt, clientConn)
		if err != nil {
			response = "ERROR: " + err.Error() + "\n"
		}

		udpRespond(conn, sender, response)
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
func udpRespond(conn net.PacketConn, sender net.Addr, txt string) {
	log.Printf("Responding to UDP with %s", txt)

	n, err := conn.WriteTo([]byte(txt), sender);
	if err != nil {
		log.Fatalf("Could not write to udp stream: %v", err)
	}
	log.Printf("WriteTo successful %d", n)
}


func tcpListener(port *string, s *sdk.SDK, cancel context.CancelFunc, clientConn *connection) {
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
		go tcpHandleConnection(conn, s, cancel, clientConn)
	}
}

// handleConnection services a single tcp connection to the server
func tcpHandleConnection(conn net.Conn, s *sdk.SDK, cancel context.CancelFunc, clientConn *connection) {
	log.Printf("TCP Client %s connected", conn.RemoteAddr().String())
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		tcpHandleCommand(conn, scanner.Text(), s, cancel, clientConn)
	}
	log.Printf("TCP Client %s disconnected", conn.RemoteAddr().String())
}

func tcpHandleCommand(conn net.Conn, txt string, s *sdk.SDK, cancel context.CancelFunc, clientConn *connection) {
	log.Printf("TCP txt: %v", txt)

	if txt == "EXIT" {
		exit(s)
	}

	response, err := handleGenAIRequest(txt, clientConn)
	if err != nil {
		response = "ERROR: " + err.Error() + "\n"
	}

	tcpRespond(conn, response)
}

// respond responds to a given sender.
func tcpRespond(conn net.Conn, txt string) {
	log.Printf("Responding to TCP with %s", txt)

	if _, err := conn.Write([]byte(txt)); err != nil {
		log.Fatalf("Could not write to TCP stream: %v", err)
	}
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

// ready attempts to mark this gameserver as ready
func ready(s *sdk.SDK) {
	err := s.Ready()
	if err != nil {
		log.Fatalf("Could not send ready message")
	}
}
