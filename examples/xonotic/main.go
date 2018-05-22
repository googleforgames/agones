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

	"agones.dev/agones/sdks/go"
)

const (
	begin        = "BEGIN"
	configLoaded = "CONFIGLOADED"
	listening    = "LISTENING"
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

// main intercepts the stdout of the Xonotic gameserver and uses it
// to determine if the game server is ready or not.
func main() {
	input := flag.String("i", "", "path to server_linux.sh")
	flag.Parse()

	fmt.Println(">>> Connecting to Agones with the SDK")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf(">>> Could not connect to sdk: %v", err)
	}

	fmt.Println(">>> Starting health checking")
	go doHealth(s)

	fmt.Println(">>> Starting wrapper for Xonotic!")
	fmt.Printf(">>> Path to Xonotic server script: %s \n", *input)

	// state tracks the state
	state := begin

	cmd := exec.Command(*input) // #nosec
	cmd.Stderr = &interceptor{forward: os.Stderr}
	cmd.Stdout = &interceptor{
		forward: os.Stdout,
		intercept: func(p []byte) {
			if state == listening {
				return
			}

			str := strings.TrimSpace(string(p))
			// since the game server starts, then loads the server.cfg
			// and then restarts itself, we need to wait for this line
			// before we look for "Server listening on address", since
			// it is output twice.
			if state == begin && str == "execing server.cfg" {
				fmt.Printf(">>> Moving to configLoaded: %s", str)
				state = configLoaded
				return
			}
			if state == configLoaded && strings.Contains(str, "Server listening on address") {
				state = listening
				fmt.Printf(">>> Moving to listening: %s", str)
				err = s.Ready()
				if err != nil {
					log.Fatalf("Could not send ready message")
				}
			}
		}}

	err = cmd.Start()
	if err != nil {
		log.Fatalf(">>> Error Starting Cmd %v", err)
	}
	err = cmd.Wait()
	log.Fatal(">>> Xonotic shutdown unexpectantly", err)
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK) {
	tick := time.Tick(2 * time.Second)
	for {
		err := sdk.Health()
		if err != nil {
			log.Fatalf("[wrapper] Could not send health ping, %v", err)
		}
		<-tick
	}
}
