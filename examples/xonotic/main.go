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
	"log"
	"time"

	"agones.dev/agones/sdks/go"
)

// main cheats a little, and just declares the game server as ready
// as soon as it fires up
func main() {
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}
	err = s.Ready()
	if err != nil {
		log.Fatalf("Could not send ready message")
	}

	doHealth(s)
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK) {
	tick := time.Tick(2 * time.Second)
	for {
		err := sdk.Health()
		if err != nil {
			log.Fatalf("Could not send health ping, %v", err)
		}
		<-tick
	}
}
