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
	"os/exec"
	"regexp"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// HELM_CMD is the command line invocation of helm
	HELM_CMD = "helm"
	// KUBECTL_CMD is the command line invocation of kubectl
	KUBECTL_CMD = "kubectl"
	// IMAGE_PULL_POLICY sets the Agones Helm configuration to always pull the image
	IMAGE_PULL_POLICY = "Always"
	// SIDECAR_PULL_POLICY sets the Agones Helm configuration to always pull the SDK image
	SIDECAR_PULL_POLICY = "true"
	// LOG_LEVEL sets the Agones Helm configuration log level
	LOG_LEVEL = "debug"
	// HELM_CHART is the helm chart for the public Agones releases
	HELM_CHART = "agones/agones"
	// AGONES_REGISTRY is the public registry for Agones releases
	AGONES_REGISTRY = "us-docker.pkg.dev/agones-images/release"
)

var (
	// TODO: Get the build version of dev (i.e. 1.44.0-dev-b765f49)
	// DEV is the current development version of Agones
	DEV = os.Getenv("DEV")
	// POD_NAME the name of the pod this container is running in
	POD_NAME = os.Getenv("POD_NAME")
	// POD_NAMESPACE the name of the pod namespace this container is running in
	POD_NAMESPACE = os.Getenv("POD_NAMESPACE")
	// VERSION_MAPPINGS are the valid Kubernetes, Agones, and Feature Gate version configurations
	VERSION_MAPPINGS = os.Getenv("version-mappings.json")
)

func main() {
	ctx := context.Background()

	validConfigs := ConfigTestSetup(ctx)
	addAgonesRepo()
	runConfigWalker(ctx, validConfigs)
}

type versionMappings struct {
	K8sToAgonesVersions       map[string][]string     `json:"k8sToAgonesVersions"`
	AgonesVersionFeatureGates map[string]featureGates `json:"agonesVersionFeatureGates"`
}

type featureGates struct {
	AlphaGates []string `json:"alphaGates"`
	BetaGates  []string `json:"betaGates"`
}

type configTest struct {
	agonesVersion string
	featureGates  string
}

type helmStatuses []struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Revision   string `json:"revision"`
	Updated    string `json:"updated"`
	Status     string `json:"status"`
	Chart      string `json:"chart"`
	AppVersion string `json:"app_version"`
}

// Determine test scenario to run
func ConfigTestSetup(ctx context.Context) []*configTest {
	versionMap := versionMappings{}

	// Find the Kubernetes version of the node that this test is running on.
	k8sVersion := findK8sVersion(ctx)

	// Get the mappings of valid Kubernetes, Agones, and Feature Gate versions from the configmap.
	err := json.Unmarshal([]byte(VERSION_MAPPINGS), &versionMap)
	if err != nil {
		log.Fatal("Could not Unmarshal", err)
	}

	// Find valid Agones versions and feature gates for the current version of Kubernetes.
	configTests := []*configTest{}
	for _, agonesVersion := range versionMap.K8sToAgonesVersions[k8sVersion] {
		ct := configTest{}
		ct.agonesVersion = agonesVersion
		if agonesVersion == "DEV" {
			ct.agonesVersion = DEV
		}
		// TODO: create different valid config based off of available feature gates
		configTests = append(configTests, &ct)
	}

	return configTests
}

// Finds the Kubernetes version of the Kubelet on the node that the current pod is running on.
// The Kubelet version is the same version as the node.
func findK8sVersion(ctx context.Context) string {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("Could not create in cluster config", cfg)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal("Could not create the kubernetes api clientset", err)
	}

	// Wait to get pod and node as these may take a while to start on a new Autopilot cluster.
	var pod *v1.Pod
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 7*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		pod, err = kubeClient.CoreV1().Pods(POD_NAMESPACE).Get(ctx, POD_NAME, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	if pod == nil || pod.Spec.NodeName == "" {
		log.Fatalf("PollUntilContextTimeout timed out. Could not get Pod: %s", err)
	}

	var node *v1.Node
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 3*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		node, err = kubeClient.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	if node == nil || node.Status.NodeInfo.KubeletVersion == "" {
		log.Fatalf("PollUntilContextTimeout timed out. Could not get Node: %s", err)
	}

	// Finds the major.min version. I.e. k8sVersion 1.30 from gkeVersion v1.30.2-gke.1587003
	gkeVersion := node.Status.NodeInfo.KubeletVersion
	log.Println("KubeletVersion", gkeVersion)
	r, err := regexp.Compile("\\d*\\.\\d*")
	if err != nil {
		log.Fatal("Could not compile regex: ", err)
	}

	return r.FindString(gkeVersion)
}

// runExecCommand executes the command with the given arguments.
func runExecCommand(cmd string, args ...string) ([]byte, error) {
	log.Println("Running command", cmd, args)
	execCommand := exec.Command(cmd, args...)

	out, err := execCommand.CombinedOutput()
	if err != nil {
		log.Printf("CombinedOutput: %s", string(out))
		log.Printf("CombinedOutput err: %s", err)
	} else {
		log.Printf("CombinedOutput: %s", string(out))
	}

	return out, err
}

// Install Agones to the cluster using the install.yaml file.
// kubectl create namespace agones-system
// kubectl apply --server-side -f https://raw.githubusercontent.com/googleforgames/agones/release-1.43.0/install/yaml/install.yaml
func installYaml() error {
	installArgs := []string{"apply", "--server-side", "-f", "https://raw.githubusercontent.com/googleforgames/agones/release-1.43.0/install/yaml/install.yaml"}

	_, err := runExecCommand(KUBECTL_CMD, installArgs...)
	return err
}

// Adds public Helm Agones releases to the cluster
func addAgonesRepo() {
	installArgs := [][]string{{"repo", "add", "agones", "https://agones.dev/chart/stable"},
		{"repo", "update"}}

	for _, args := range installArgs {
		_, err := runExecCommand(HELM_CMD, args...)
		if err != nil {
			log.Fatalf("Could not add Agones helm repo: %s", err)
		}
	}
}

func installAgonesRelease(version, registry, featureGates, imagePullPolicy, sidecarPullPolicy,
	logLevel, chart string) error {

	// TODO: Include feature gates. (Current issue with Helm and string formatting of the feature gates.)
	// 		"--set agones.featureGates=%s "+

	helmString := fmt.Sprintf(
		"upgrade --install --atomic --wait --timeout=10m --namespace=agones-system --create-namespace --version %s "+
			"--set agones.image.tag=%s "+
			"--set agones.image.registry=%s "+
			"--set agones.image.allocator.pullPolicy=%s "+
			"--set agones.image.controller.pullPolicy=%s "+
			"--set agones.image.extensions.pullPolicy=%s "+
			"--set agones.image.ping.pullPolicy=%s "+
			"--set agones.image.sdk.alwaysPull=%s "+
			"--set agones.controller.logLevel=%s "+
			"agones %s",
		version, version, registry, imagePullPolicy, imagePullPolicy, imagePullPolicy, imagePullPolicy,
		sidecarPullPolicy, logLevel, chart,
	)

	helmArgs := strings.Split(helmString, " ")

	_, err := runExecCommand(HELM_CMD, helmArgs...)
	if err != nil {
		return err
	}

	return err
}

func runConfigWalker(ctx context.Context, validConfigs []*configTest) {
	for _, config := range validConfigs {
		registry := AGONES_REGISTRY
		chart := HELM_CHART
		if config.agonesVersion == DEV {
			// TODO: Update to templated value for registry and chart for Dev build
			continue
		}

		err := installAgonesRelease(config.agonesVersion, registry, config.featureGates, IMAGE_PULL_POLICY,
			SIDECAR_PULL_POLICY, LOG_LEVEL, chart)
		if err != nil {
			log.Printf("installAgonesRelease err: %s", err)
		}

		// Wait for the helm release to install. Waits the same amount of time as the Helm timeout.
		var helmStatus string
		err = wait.PollUntilContextTimeout(ctx, 10*time.Second, 10*time.Minute, true, func(ctx context.Context) (done bool, err error) {
			helmStatus = checkHelmStatus(config.agonesVersion)
			if helmStatus == "deployed" {
				return true, nil
			}
			return false, nil
		})

		if err != nil || helmStatus != "deployed" {
			log.Fatalf("PollUntilContextTimeout timed out while attempting upgrade to Agones version %s. Helm Status %s",
				config.agonesVersion, helmStatus)
		}
	}
}

// checkHelmStatus returns the status of the Helm release at a specified agonesVersion if it exists.
func checkHelmStatus(agonesVersion string) string {
	helmStatus := helmStatuses{}
	checkStatus := []string{"list", "-a", "-nagones-system", "-ojson"}
	out, err := runExecCommand(HELM_CMD, checkStatus...)
	if err != nil {
		log.Fatalf("Could not run command %s %s, err: %s", KUBECTL_CMD, checkStatus, err)
	}

	err = json.Unmarshal(out, &helmStatus)
	if err != nil {
		log.Fatal("Could not Unmarshal", err)
	}

	for _, status := range helmStatus {
		if status.AppVersion == agonesVersion {
			return status.Status
		}
	}
	return ""
}
