// Copyright 2020 Google LLC All Rights Reserved.
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
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	sdk "agones.dev/agones/sdks/go"
)

// logLocation is the path to the location of the SuperTuxKart log file
const logLocation = "/.config/supertuxkart/config-0.10/server_config.log"

// main intercepts the log file of the SuperTuxKart gameserver and uses it
// to determine if the game server is ready or not.
func main() {
	input := flag.String("i", "", "the command and arguments to execute the server binary")
	flag.Parse()

	fmt.Println(">>> Connecting to Agones with the SDK")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf(">>> Could not connect to SDK: %v", err)
	}

	fmt.Println(">>> Starting health checking")
	go doHealth(s)

	fmt.Println(">>> Starting wrapper for SuperTuxKart")
	fmt.Printf(">>> Command being run for SuperTuxKart server: %s \n", *input)

	cmdString := strings.Split(*input, " ")
	command, args := cmdString[0], cmdString[1:]

	cmd := exec.Command(command, args...) // #nosec
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		log.Fatalf(">>> Error starting cmd: %v", err)
	}

	// SuperTuxKart refuses to output to foreground, so we're going to
	// poll the server log.
	ready := false
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf(">>> Could not get home dir: %v", err)
	}
	for i := 0; i <= 10; i++ {
		time.Sleep(time.Second)

		logFile, err := os.Open(path.Join(home, logLocation))
		if err != nil {
			log.Printf(">>> Could not open server log: %v", err)
			continue
		}

		content, err := ioutil.ReadAll(logFile)
		if err != nil {
			log.Printf(">>> Could not read server file: %v", err)
			continue
		}
		output := string(content)
		println(">>> Output from server_config.log:")
		print(output)

		if strings.Contains(output, "Listening has been started") {
			ready = true
			log.Printf(">>> Moving to READY!")
			if err := s.Ready(); err != nil {
				log.Fatalf(">>> Could not send ready message")
			}
			break
		}
	}
	if ready == false {
		log.Fatalf(">>> Server did not become ready within 10 seconds")
	}

	err = cmd.Wait()
	log.Fatalf(">>> SuperTuxKart shutdown unexpectedly: %v", err)
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK) {
	tick := time.Tick(2 * time.Second)
	for {
		if err := sdk.Health(); err != nil {
			log.Fatalf(">>> Could not send health ping: %v", err)
		}
		<-tick
	}
}
