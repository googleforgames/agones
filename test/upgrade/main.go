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

// Package main is controller for running the Agones in-place upgrades tests
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	validConfigs := ConfigTestSetup()

	// TODO(igooch): Replace this with the deterministic config walker
	for i, validConfig := range validConfigs {
		log.Printf("validConfig %d: %v", i+1, validConfig)
	}
}

var (
	// DEV is the current development version of Agones
	DEV = os.Getenv("DEV")
	// POD_NAME the name of the pod this container is running in
	POD_NAME = os.Getenv("POD_NAME")
	// POD_NAMESPACE the name of the pod namespace this container is running in
	POD_NAMESPACE = os.Getenv("POD_NAMESPACE")
	// VERSION_MAPPINGS are the valid Kubernetes, Agones, and Feature Gate version configurations
	VERSION_MAPPINGS = os.Getenv("version-mappings.json")
)

type versionMappings struct {
	K8sToAgonesVersions       map[string][]string     `json:"k8sToAgonesVersions"`
	AgonesVersionFeatureGates map[string]featureGates `json:"agonesVersionFeatureGates"`
}

type featureGates struct {
	AlphaGates []string `json:"alphaGates"`
	BetaGates  []string `json:"betaGates"`
}

type configTest struct {
	k8sVersion    string
	agonesVersion string
	featureGates  string
}

// Determine test scenario to run
func ConfigTestSetup() []*configTest {
	versionMap := versionMappings{}

	// Find the Kubernetes version of the node that this test is running on.
	k8sVersion := findK8sVersion()

	// Get the mappings of valid Kubernetes, Agones, and Feature Gate versions from the configmap.
	err := json.Unmarshal([]byte(VERSION_MAPPINGS), &versionMap)
	if err != nil {
		log.Fatal("Could not Unmarshal", err)
	}

	// Find valid Agones versions and feature gates for the current version of Kubernetes.
	configTests := []*configTest{}
	for _, agonesVersion := range versionMap.K8sToAgonesVersions[k8sVersion] {
		ct := configTest{}
		ct.k8sVersion = k8sVersion
		ct.agonesVersion = agonesVersion
		ct.featureGates = defaultGates(versionMap.AgonesVersionFeatureGates[agonesVersion])
		configTests = append(configTests, &ct)
	}

	return configTests
}

// Finds the Kubernetes version of the Kubelet on the node that the current pod is running on.
// The Kubelet version is the same version as the node.
func findK8sVersion() string {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("Could not create in cluster config", cfg)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal("Could not create the kubernetes api clientset", err)
	}

	ctx := context.Background()

	pod, err := kubeClient.CoreV1().Pods(POD_NAMESPACE).Get(ctx, POD_NAME, metav1.GetOptions{})
	if err != nil {
		log.Fatal("Could not get pod", err)
	}

	node, err := kubeClient.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		log.Fatal("Could not get node", err)
	}

	// Finds the major.min version. I.e. k8sVersion 1.30 from gkeVersion v1.30.2-gke.1587003
	gkeVersion := node.Status.NodeInfo.KubeletVersion
	log.Println("KubeletVersion", gkeVersion)
	r, err := regexp.Compile("\\d*\\.\\d*")
	if err != nil {
		log.Fatal("Could not compile regex", err)
	}

	return r.FindString(gkeVersion)
}

// Takes in featureGates struct and returns a string of the default feature gates that can be passed
// to the Agones installation command line argument `featureWithGate=`
func defaultGates(fg featureGates) string {
	var featureWithGate strings.Builder

	for i, featureGate := range fg.AlphaGates {
		// Only append the & if there is another featureGate following the current featureGate
		if i < (len(fg.AlphaGates)-1) || len(fg.BetaGates) != 0 {
			fmt.Fprintf(&featureWithGate, "%s=false&", featureGate)
		} else {
			fmt.Fprintf(&featureWithGate, "%s=false", featureGate)
		}
	}

	for i, featureGate := range fg.BetaGates {
		if i < (len(fg.BetaGates) - 1) {
			fmt.Fprintf(&featureWithGate, "%s=true&", featureGate)
		} else {
			fmt.Fprintf(&featureWithGate, "%s=true", featureGate)
		}
	}

	return featureWithGate.String()
}
