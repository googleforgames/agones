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
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	sdk "agones.dev/agones/sdks/go"
)

type interceptor struct {
	forward   io.Writer
	intercept func(p []byte)
}

// Write will intercept the incoming stream, and forward
// the contents to its `forward` Writer.
func (i *interceptor) Write(p []byte) (n int, err error) {
	if i.intercept != nil {
		i.intercept(p)
	}

	return i.forward.Write(p)
}

// main intercepts the stdout of the Minetest gameserver and uses it
// to determine if the game server is ready or not.
func main() {
	input := flag.String("i", "", "path to minetestserver.sh")
	args := flag.String("args", "", "additional arguments to pass to the script")
	flag.Parse()

	argsList := strings.Split(strings.Trim(strings.TrimSpace(*args), "'"), " ")
	fmt.Println(">>> Connecting to Agones with the SDK")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf(">>> Could not connect to sdk: %v", err)
	}

	fmt.Println(">>> Starting health checking")
	go doHealth(s)

	fmt.Println(">>> Starting wrapper for Minetest!")
	fmt.Printf(">>> Path to Minetest server script: %s %v\n", *input, argsList)

	// track references to listening count
	listeningCount := 0

	cmd := exec.Command(*input, argsList...) // #nosec
	cmd.Stderr = &interceptor{forward: os.Stderr}
	cmd.Stdout = &interceptor{
		forward: os.Stdout,
		intercept: func(p []byte) {
			if listeningCount >= 1 {
				return
			}

			str := strings.TrimSpace(string(p))
			// Minetest will say "listening on 0.0.0.0:30000" when ready.
			if count := strings.Count(str, "listening on 0.0.0.0:30000"); count > 0 {
				listeningCount += count
				fmt.Printf(">>> Found 'listening' statement: %d \n", listeningCount)

				if listeningCount >= 1 {
					fmt.Printf(">>> Moving to READY: %s \n", str)
					err = s.Ready()
					if err != nil {
						log.Fatalf("Could not send ready message")
					}
				}
			}
		}}

	if err := cmd.Start(); err != nil {
		log.Fatalf(">>> Error Starting Cmd %v", err)
	}
	err = cmd.Wait()
	log.Fatal(">>> Minetest shutdown unexpectantly", err)
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK) {
	tick := time.Tick(2 * time.Second)
	for {
		if err := sdk.Health(); err != nil {
			log.Fatalf("[wrapper] Could not send health ping, %v", err)
		}
		<-tick
	}
}
