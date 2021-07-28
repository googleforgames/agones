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
	"context"
	"flag"
	"os/user"
	"path/filepath"
	"sync"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	e2eframework "agones.dev/agones/test/e2e/framework"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultNs = "default"
const reqPerClient = 10

func main() {
	usr, err := user.Current()
	if err != nil {
		logrus.Fatalf("Unable to determine the current user: %v", err)
	}
	kubeconfig := flag.String("kubeconfig", filepath.Join(usr.HomeDir, ".kube", "config"),
		"kube config path, e.g. $HOME/.kube/config")
	fleetName := flag.String("fleet_name", "simple-game-server", "The fleet name that the tests will run against")
	qps := flag.Int("qps", 1000, "The QPS value that will overwrite the default value")
	burst := flag.Int("burst", 1000, "The Burst value that will overwrite the default value")
	clientCnt := flag.Int("clients", 10, "The number of concurrent clients")

	flag.Parse()

	logrus.SetFormatter(&logrus.TextFormatter{
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		TimestampFormat:           "2006-01-02 15:04:05.000",
	})

	framework, err := e2eframework.NewWithRates(*kubeconfig, float32(*qps), *burst)
	if err != nil {
		logrus.Fatalf("Failed to setup framework: %v", err)
	}

	logrus.Info("Starting Allocation")
	allocate(framework, *clientCnt, *fleetName)
	logrus.Info("Finished Allocation.")
	logrus.Info("=======================================================================")
	logrus.Info("=======================================================================")
	logrus.Info("=======================================================================")
}

func allocate(framework *e2eframework.Framework, numOfClients int, fleetName string) {
	gsa := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: allocationv1.GameServerAllocationSpec{
			Required: allocationv1.GameServerSelector{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: fleetName}}},
			Preferred: []allocationv1.GameServerSelector{
				{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: fleetName}}},
			},
		},
	}
	var wg sync.WaitGroup
	wg.Add(numOfClients)

	// Allocate GS by numOfClients in parallel
	for i := 0; i < numOfClients; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < reqPerClient; j++ {
				gsa1, err := framework.AgonesClient.AllocationV1().GameServerAllocations(defaultNs).Create(context.Background(), gsa.DeepCopy(), metav1.CreateOptions{})
				if err != nil {
					logrus.Errorf("could not completed gsa1 allocation : %v", err)
				} else if gsa1.Status.State == "Contention" {
					logrus.Errorf("could not allocate : %v", gsa1.Status.State)
				}
				logrus.Infof("%+v", gsa1)
			}
		}()
	}

	wg.Wait()
}
