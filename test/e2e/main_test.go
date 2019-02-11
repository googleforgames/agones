// Copyright 2018 Google Inc. All Rights Reserved.
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

package e2e

import (
	"flag"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	e2eframework "agones.dev/agones/test/e2e/framework"
	"github.com/sirupsen/logrus"
)

const defaultNs = "default"

var framework *e2eframework.Framework

func TestMain(m *testing.M) {
	usr, _ := user.Current()
	kubeconfig := flag.String("kubeconfig", filepath.Join(usr.HomeDir, "/.kube/config"),
		"kube config path, e.g. $HOME/.kube/config")
	gsimage := flag.String("gameserver-image", "gcr.io/agones-images/udp-server:0.6",
		"gameserver image to use for those tests, gcr.io/agones-images/udp-server:0.6")
	pullSecret := flag.String("pullsecret", "",
		"optional secret to be used for pulling the gameserver and/or Agones SDK sidecar images")
	stressTestLevel := flag.Int("stress", 0, "enable stress test at given level 0-100")

	flag.Parse()

	logrus.SetFormatter(&logrus.TextFormatter{
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		TimestampFormat:           "2006-01-02 15:04:05.000",
	})

	var (
		err      error
		exitCode int
	)

	if framework, err = e2eframework.New(*kubeconfig, *gsimage, *pullSecret, *stressTestLevel); err != nil {
		log.Printf("failed to setup framework: %v\n", err)
		os.Exit(1)
	}

	// run cleanup before tests, to ensure no resources from previous runs exist.
	err = framework.CleanUp(defaultNs)
	if err != nil {
		log.Printf("failed to cleanup resources: %v\n", err)
	}

	defer func() {
		err = framework.CleanUp(defaultNs)
		if err != nil {
			log.Printf("failed to cleanup resources: %v\n", err)
		}
		os.Exit(exitCode)
	}()
	exitCode = m.Run()

}
