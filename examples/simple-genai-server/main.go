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
	"sync"
	"time"

	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
)

// Main starts a server that serves as an example of how to integrate GenAI endpoints into your dedicated game server.
func main() {
	sigCtx, _ := signals.NewSigKillContext()

	port := flag.String("port", "7654", "The port to listen to traffic on")
	genAiEndpoint := flag.String("GenAiEndpoint", "", "The full base URL to send API requests to to simulate computer (NPC) responses to user input")
	genAiContext := flag.String("GenAiContext", "", "Context for the GenAI endpoint")
	prompt := flag.String("Prompt", "", "The first prompt for the GenAI endpoint")
	simEndpoint := flag.String("SimEndpoint", "", "The full base URL to send API requests to to simulate user input")
	simContext := flag.String("SimContext", "", "Context for the Sim endpoint")
	numChats := flag.Int("NumChats", 1, "Number of back and forth chats between the sim and genAI")

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
	}
	if gae := os.Getenv("GenAiEndpoint"); gae != "" {
		genAiEndpoint = &gae
	}
	if nc := os.Getenv("NumChats"); nc != "" {
		num, err := strconv.Atoi(nc)
		if err != nil {
			log.Fatalf("Could not parse NumChats: %v", err)
		}
		numChats = &num
	}

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	log.Print("Starting Health Ping")
	go doHealth(s, sigCtx)

	var simConn *connection
	if *simEndpoint != "" {
		log.Printf("Creating Sim Client at endpoint %s", *simEndpoint)
		simConn = initClient(*simEndpoint, *simContext, "Sim")
	}

	if *genAiEndpoint == "" {
		log.Fatalf("GenAiEndpoint must be specified")
	}
	log.Printf("Creating GenAI Client at endpoint %s", *genAiEndpoint)
	genAiConn := initClient(*genAiEndpoint, *genAiContext, "GenAI")

	log.Print("Marking this server as ready")
	if err := s.Ready(); err != nil {
		log.Fatalf("Could not send ready message")
	}

	// Start up TCP listener so the user can interact with the GenAI endpoint manually
	if simConn == nil {
		go tcpListener(*port, genAiConn)
		<-sigCtx.Done()
	} else {
		log.Printf("Starting autonomous chat with Prompt: %s", *prompt)
		var wg sync.WaitGroup
		// TODO: Add flag for creating X number of chats
		wg.Add(1)
		go autonomousChat(*prompt, genAiConn, simConn, *numChats, &wg, sigCtx)
		wg.Wait()
	}

	log.Printf("Shutting down the Game Server.")
	shutdownErr := s.Shutdown()
	if shutdownErr != nil {
		log.Printf("Could not shutdown")
	}
	os.Exit(0)
}

func initClient(endpoint string, context string, name string) *connection {
	// TODO: create option for a client certificate
	client := &http.Client{}
	return &connection{client: client, endpoint: endpoint, context: context, name: name}
}

type connection struct {
	client   *http.Client
	endpoint string // full base URL for API requests
	context  string
	name     string // human readable name for the connection
	// TODO: create options for routes off the base URL
}

type GenAIRequest struct {
	Context string `json:"context,omitempty"`
	Prompt  string `json:"prompt"`
}

func handleGenAIRequest(prompt string, clientConn *connection) (string, error) {
	jsonRequest := GenAIRequest{
		Context: clientConn.context,
		Prompt:  prompt,
	}
	jsonStr, err := json.Marshal(jsonRequest)
	if err != nil {
		return "", fmt.Errorf("unable to marshal json request: %v", err)
	}

	req, err := http.NewRequest("POST", clientConn.endpoint, bytes.NewBuffer(jsonStr))
	if err != nil {
		return "", fmt.Errorf("unable create http POST request: %v", err)
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
func autonomousChat(prompt string, conn1 *connection, conn2 *connection, numChats int, wg *sync.WaitGroup, sigCtx context.Context) {
	select {
	case <-sigCtx.Done():
		wg.Done()
		return
	default:
		if numChats <= 0 {
			wg.Done()
			return
		}

		response, err := handleGenAIRequest(prompt, conn1)
		if err != nil {
			log.Fatalf("Could not send request: %v", err)
		}
		log.Printf("%d %s RESPONSE: %s\n", numChats, conn1.name, response)

		numChats -= 1
		// Flip between the connection that the response is sent to.
		autonomousChat(response, conn2, conn1, numChats, wg, sigCtx)
	}
}

// Manually interact via TCP with the GenAI endpont
func tcpListener(port string, genAiConn *connection) {
	log.Printf("Starting TCP server, listening on port %s", port)
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Could not start TCP server: %v", err)
	}
	defer ln.Close() // nolint: errcheck

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalf("Unable to accept incoming TCP connection: %v", err)
		}
		go tcpHandleConnection(conn, genAiConn)
	}
}

// handleConnection services a single tcp connection to the GenAI endpoint
func tcpHandleConnection(conn net.Conn, genAiConn *connection) {
	log.Printf("TCP Client %s connected", conn.RemoteAddr().String())

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		txt := scanner.Text()
		log.Printf("TCP txt: %v", txt)

		response, err := handleGenAIRequest(txt, genAiConn)
		if err != nil {
			response = "ERROR: " + err.Error() + "\n"
		}

		if _, err := conn.Write([]byte(response)); err != nil {
			log.Fatalf("Could not write to TCP stream: %v", err)
		}
	}

	log.Printf("TCP Client %s disconnected", conn.RemoteAddr().String())
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
