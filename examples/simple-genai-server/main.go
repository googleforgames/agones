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
	"strconv"
	"time"

	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
)

// Main starts a server that serves as an example of how to integrate GenAI endpoints into your dedicated game server.
func main() {
	sigCtx, _ := signals.NewSigKillContext()

	port := flag.String("port", "7654", "The port to listen to traffic on")
	simEndpoint := flag.String("simEndpoint", "", "The full base URL to send API requests to to simulate user input")
	genAiEndpoint := flag.String("genAiEndpoint", "", "The full base URL to send API requests to to simulate computer (NPC) responses to user input")
	prompt := flag.String("Prompt", "", "The first prompt for the GenAI endpoint")
	simContext := flag.String("SimContext", "", "Context for the Sim endpoint")
	genAiContext := flag.String("GenAiContext", "", "Context for the GenAI endpoint")
	numChats := flag.Int("NumChats", 1, "Number of back and forth chats between the sim and genAI")

	var simConn *connection
	var compConn * connection

	flag.Parse()
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	if sc := os.Getenv("SimContext"); sc != "" {
		simContext = &sc
	}
	if gac := os.Getenv("GenAiContext"); gac != "" {
		genAiContext = &gac
	}
	if p := os.Getenv("Prompt"); p != "" {
		prompt = &p
	}
	if se := os.Getenv("SimEndpoint"); se != "" {
		simEndpoint = &se
		log.Printf("Creating Client at endpoint %s", *simEndpoint)
		simConn = initClient(*simEndpoint, *simContext)
	}
	if gae := os.Getenv("GenAiEndpoint"); gae != "" {
		genAiEndpoint = &gae
		log.Printf("Creating Client at endpoint %s", *genAiEndpoint)
		compConn = initClient(*genAiEndpoint, *genAiContext)
	}
	if nc := os.Getenv("NumChats"); nc != "" {
		num, err := strconv.Atoi(nc)
		numChats = &num
		if err != nil {
			log.Fatalf("Could not parse NumChats: %v", err)
		}
	}

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	log.Print("Starting Health Ping")
	// TODO: Is "cancel" still being used even if no method explicitly uses it?
	ctx, _ := signals.NewSigKillContext()
	go doHealth(s, ctx)

	// Start up TCP listener so the user can interact with the GenAI endpoint manually
	if simConn == nil {
		go tcpListener(port, s, compConn)
	} else {
		go handleChat(*prompt, compConn, simConn, *numChats)
	}


	log.Print("Marking this server as ready")
	if err := s.Ready(); err != nil {
		log.Fatalf("Could not send ready message")
	}

	<-sigCtx.Done()
	os.Exit(0)
}

func initClient(endpoint string, context string) (*connection) {
	// TODO: create option for a client certificate
	client := &http.Client{}
	conn := &connection{client: client, endpoint: endpoint, context: context}
	return conn
}

type connection struct {
	client   *http.Client
	endpoint string // full base URL for API requests
	context  string
	// TODO: create options for routes off the base URL
}

type GenAIRequest struct {
	Context         string  `json:"context,omitempty"`
	MaxOutputTokens int     `json:"max_output_tokens"`
	Prompt          string  `json:"prompt"`
	Temperature     float64 `json:"temperature"`
	TopK            int     `json:"top_k"`
	TopP            float64 `json:"top_p"`
}

func handleGenAIRequest(prompt string, clientConn *connection) (string, error) {
	jsonRequest := GenAIRequest{
		Context: clientConn.context,
		MaxOutputTokens: 256,
		Prompt: prompt,
		Temperature: 0.2,
		TopK: 40,
		TopP: 0.8,
	}
	jsonStr, err := json.Marshal(jsonRequest)
	if err != nil {
		return "", fmt.Errorf("unable to marshal json request: %v", err)
	}

	req, err := http.NewRequest("POST", clientConn.endpoint, bytes.NewBuffer(jsonStr))
	if err != nil {
		return "unable create http POST request", err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := clientConn.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to post request: %v", err)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read response body: %v", err)
	}
	defer resp.Body.Close()
	body := string(responseBody)

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Status: %s, Body: %s", resp.Status, body)
	}
	return string(responseBody) + "\n", err
}

// Two AIs (connection endpoints) talking to each other
func handleChat(prompt string, conn1 *connection, conn2 *connection, numChats int) {
	if numChats <= 0 {
		return
	}
	response, err := handleGenAIRequest(prompt, conn1)
	if err != nil {
		log.Fatalf("could not send request: %v", err)
	} else {
		log.Printf("%d PROMPT: %s\nRESPONSE: %s\n", numChats, prompt, response)
	}

	numChats -= 1
	handleChat(response, conn2, conn1, numChats)
}

func tcpListener(port *string, s *sdk.SDK, clientConn *connection) {
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
		go tcpHandleConnection(conn, s, clientConn)
	}
}

// handleConnection services a single tcp connection to the server
func tcpHandleConnection(conn net.Conn, s *sdk.SDK, clientConn *connection) {
	log.Printf("TCP Client %s connected", conn.RemoteAddr().String())
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		tcpHandleCommand(conn, scanner.Text(), s, clientConn)
	}
	log.Printf("TCP Client %s disconnected", conn.RemoteAddr().String())
}

func tcpHandleCommand(conn net.Conn, txt string, s *sdk.SDK, clientConn *connection) {
	log.Printf("TCP txt: %v", txt)

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
