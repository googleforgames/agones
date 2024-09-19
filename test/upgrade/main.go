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
	"html/template"
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
	// HelmCmd is the command line invocation of helm
	HelmCmd = "helm"
	// KubectlCmd is the command line invocation of kubectl
	KubectlCmd = "kubectl"
	// ImagePullPolicy sets the Agones Helm configuration to always pull the image
	ImagePullPolicy = "Always"
	// SidecarPullPolicy sets the Agones Helm configuration to always pull the SDK image
	SidecarPullPolicy = "true"
	// LogLevel sets the Agones Helm configuration log level
	LogLevel = "debug"
	// HelmChart is the helm chart for the public Agones releases
	HelmChart = "agones/agones"
	// AgonesRegistry is the public registry for Agones releases
	AgonesRegistry = "us-docker.pkg.dev/agones-images/release"
	// TestRegistry is the public registry for upgrade test container files
	// TODO: Create Test Registry in agones-images/ci
	TestRegistry = "us-docker.pkg.dev/agones-images/ci/sdk-client-test"
)

var (
	// Dev is the current development version of Agones
	// TODO: Get the build version of dev (i.e. 1.44.0-dev-b765f49)
	Dev = os.Getenv("Dev")
	// ReleaseVersion is the latest released version of Agones (DEV - 1).
	ReleaseVersion = os.Getenv("ReleaseVersion")
	// PodName the name of the pod this container is running in
	PodName = os.Getenv("PodName")
	// PodNamespace the name of the pod namespace this container is running in
	PodNamespace = os.Getenv("PodNamespace")
	// VersionMappings are the valid Kubernetes, Agones, and Feature Gate version configurations
	VersionMappings = os.Getenv("version-mappings.json")
)

func main() {
	ctx := context.Background()

	validConfigs := configTestSetup(ctx)
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
	agonesVersion  string
	featureGates   string
	gameServerPath string
}

type gameServerTemplate struct {
	AgonesVersion string
	Registry      string
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
func configTestSetup(ctx context.Context) []*configTest {
	versionMap := versionMappings{}

	// Find the Kubernetes version of the node that this test is running on.
	k8sVersion := findK8sVersion(ctx)

	// Get the mappings of valid Kubernetes, Agones, and Feature Gate versions from the configmap.
	err := json.Unmarshal([]byte(VersionMappings), &versionMap)
	if err != nil {
		log.Fatal("Could not Unmarshal", err)
	}

	// Find valid Agones versions and feature gates for the current version of Kubernetes.
	configTests := []*configTest{}
	for _, agonesVersion := range versionMap.K8sToAgonesVersions[k8sVersion] {
		ct := configTest{}
		ct.agonesVersion = agonesVersion
		if agonesVersion == "DEV" {
			ct.agonesVersion = Dev
			// Game server container cannot be created at DEV version due to go.mod only able to access
			// published Agones versions. Use N-1 for DEV.
			ct.gameServerPath = createGameServerFile(ReleaseVersion)
		} else {
			ct.gameServerPath = createGameServerFile(agonesVersion)
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
		pod, err = kubeClient.CoreV1().Pods(PodNamespace).Get(ctx, PodName, metav1.GetOptions{})
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
	r := regexp.MustCompile(`\d+\.\d+`)

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

// Adds public Helm Agones releases to the cluster
func addAgonesRepo() {
	installArgs := [][]string{{"repo", "add", "agones", "https://agones.dev/chart/stable"},
		{"repo", "update"}}

	for _, args := range installArgs {
		_, err := runExecCommand(HelmCmd, args...)
		if err != nil {
			log.Fatalf("Could not add Agones helm repo: %s", err)
		}
	}
}

func installAgonesRelease(version, registry, featureGates, imagePullPolicy, sidecarPullPolicy,
	logLevel, chart string) error {

	// TODO: Include feature gates. (Current issue with Helm and string formatting of the feature gates.)
	// 		"--set agones.featureGates=%s "+
	// Remove this print line which is here to prevent linter complaining about featureGates not used.
	log.Printf("Agones Version %s, FeatureGates %s", version, featureGates)

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

	_, err := runExecCommand(HelmCmd, helmArgs...)
	if err != nil {
		return err
	}

	return err
}

func runConfigWalker(ctx context.Context, validConfigs []*configTest) {
	// Done channel ensures that the createGameServers starts after the Agones version has been
	// installed runs until the next Agones version has been installed.
	done := make(chan bool)

	for _, config := range validConfigs {
		registry := AgonesRegistry
		chart := HelmChart
		if config.agonesVersion == Dev {
			// TODO: Update to templated value for registry and chart for Dev build
			continue
		}
		err := installAgonesRelease(config.agonesVersion, registry, config.featureGates, ImagePullPolicy,
			SidecarPullPolicy, LogLevel, chart)
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

		// Install Gameservers at the existing version.
		// Start multiple GameServers (start one every X seconds).
		// Each GameServer shuts itself down when done (part of the sdk-client-test.go)
		// Run concurrently (do not block surrounding for loop)
		// Run until we reach this line of code again (the next version of Agones has been installed).
		// TODO:
		// Create Watcher if the GameServer fails, (log.Fatalf), log info (current Agones version, Game
		//  server version, what Agones version is being installed) and fail this test and end job.
		close(done)
		done = make(chan bool)
		go createGameServers(done, config.gameServerPath)
	}
	// TODO:
	// Wait for the existing Game Servers finish naturally by reaching their shutdown phase.
	time.Sleep(2 * time.Minute)
	close(done)
}

// checkHelmStatus returns the status of the Helm release at a specified agonesVersion if it exists.
func checkHelmStatus(agonesVersion string) string {
	helmStatus := helmStatuses{}
	checkStatus := []string{"list", "-a", "-nagones-system", "-ojson"}
	out, err := runExecCommand(HelmCmd, checkStatus...)
	if err != nil {
		log.Fatalf("Could not run command %s %s, err: %s", KubectlCmd, checkStatus, err)
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

// Creates a gameserver yaml file from the mounted gameserver.yaml template. The name of the new
// gameserver yaml is based on the Agones version, i.e. gs1440.yaml for Agones version 1.44.0
func createGameServerFile(agonesVersion string) string {
	gsTmpl := gameServerTemplate{Registry: TestRegistry, AgonesVersion: agonesVersion}

	gsTemplate, err := template.ParseFiles("gameserver.yaml")
	if err != nil {
		log.Fatal("Could not ParseFiles template gameserver.yaml", err)
	}

	// Must use /tmp since the container is a non-root user.
	gsPath := strings.ReplaceAll("/tmp/gs"+agonesVersion, ".", "")
	gsPath += ".yaml"
	// Check if the file already exists
	if _, err := os.Stat(gsPath); err == nil {
		return gsPath
	}
	gsFile, err := os.Create(gsPath)
	if err != nil {
		log.Fatal("Could not create file ", err)
	}

	err = gsTemplate.Execute(gsFile, gsTmpl)
	if err != nil {
		if fErr := gsFile.Close(); fErr != nil {
			log.Printf("Could not close game server file %s, err: %s", gsPath, fErr)
		}
		log.Fatal("Could not Execute template gameserver.yaml ", err)
	}
	if err = gsFile.Close(); err != nil {
		log.Printf("Could not close game server file %s, err: %s", gsPath, err)
	}

	return gsPath
}

// Create a game server every two seconds until the channel is closed.
func createGameServers(done chan bool, gsPath string) {
	args := []string{"create", "-f", gsPath}
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-done:
			ticker.Stop()
			return
		case <-ticker.C:
			_, err := runExecCommand(KubectlCmd, args...)
			// TODO: Do not ignore error if unable to create due to something other than cluster scale up
			if err != nil {
				// log.Fatalf("Could not create Gameserver %s: %s", gsPath, err)
				log.Printf("Could not create Gameserver %s: %s", gsPath, err)
			}
		}
	}
}
