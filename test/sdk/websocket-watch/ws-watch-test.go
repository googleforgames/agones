// Copyright 2021 Google LLC All Rights Reserved.
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
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	portStr := os.Getenv("AGONES_SDK_HTTP_PORT")
	watchURL := url.URL{Scheme: "ws", Host: "localhost:" + portStr, Path: "/watch/gameserver"}
	readyURL := url.URL{Scheme: "http", Host: "localhost:" + portStr, Path: "/metadata/label"}
	log.Printf("Connecting to %s", watchURL.String())
	websocketClient, connectResponse, dialErr := websocket.DefaultDialer.Dial(watchURL.String(), nil)
	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}

	if dialErr != nil {
		log.Fatal("Could not dial watch websocket:", dialErr)
	}

	defer connectResponse.Body.Close() // nolint: errcheck

	defer websocketClient.Close() // nolint: errcheck

	done := make(chan struct{})

	go func() {
		defer close(done) // nolint: errcheck
		_, message, err := websocketClient.ReadMessage()
		if err != nil {
			log.Fatalf("Unable to read message from websocket: %s", err)
			return
		}
		log.Printf("Received message from websocket: %s", message)

		if strings.Contains(string(message), "agones.dev/sdk-testws") {
			log.Printf("Found label 'agones.dev/sdk-testws' in message")
		} else {
			log.Fatalf("Could not find label 'agones.dev/sdk-testws' in message")
		}
		done <- struct{}{}
	}()

	timeout := time.NewTicker(time.Second)
	defer timeout.Stop()

	tries := 0

	req, reqErr := http.NewRequest("PUT", readyURL.String(), strings.NewReader("{\"key\": \"testws\", \"value\": \"true\"}"))

	if reqErr != nil {
		log.Fatalf("Could not create label request: %s", reqErr) // nolint: gocritic
	}

	response, respErr := httpClient.Do(req)

	if respErr != nil {
		log.Fatalf("Could not put label request: %s", reqErr) // nolint: gocritic
	}

	defer response.Body.Close() // nolint: errcheck

L:
	for {
		select {
		case <-done:
			break L
		case <-timeout.C:
			if tries > 10 {
				log.Fatal("Test timed out")
			}
			tries++
		}
	}

	closeErr := websocketClient.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	if closeErr != nil {
		log.Fatalf("Error writing close message: %s", closeErr)
	}
}
