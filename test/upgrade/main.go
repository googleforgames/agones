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
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
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
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("Could not create in cluster config", cfg)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal("Could not create the kubernetes api clientset", err)
	}

	stopWatch := make(chan struct{})
	go watchGameServerPods(kubeClient, stopWatch)

	validConfigs := configTestSetup(ctx, kubeClient)
	addAgonesRepo()
	runConfigWalker(ctx, validConfigs)

	close(stopWatch)
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
	fleetPath     string
}

// CountsAndLists can be removed from the template once CountsAndLists is GA in all tested versions
type gameServerTemplate struct {
	AgonesVersion  string
	Registry       string
	CountsAndLists bool
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
func configTestSetup(ctx context.Context, kubeClient *kubernetes.Clientset) []*configTest {
	versionMap := versionMappings{}

	// Find the Kubernetes version of the node that this test is running on.
	k8sVersion := findK8sVersion(ctx, kubeClient)

	// Get the mappings of valid Kubernetes, Agones, and Feature Gate versions from the configmap.
	err := json.Unmarshal([]byte(VersionMappings), &versionMap)
	if err != nil {
		log.Fatal("Could not Unmarshal", err)
	}

	// Find valid Agones versions and feature gates for the current version of Kubernetes.
	configTests := []*configTest{}
	for _, agonesVersion := range versionMap.K8sToAgonesVersions[k8sVersion] {
		ct := configTest{}
		// TODO: create different valid config based off of available feature gates. containsCountsAndLists
		// will need to be updated to return true for when CountsAndLists=true.
		countsAndLists := containsCountsAndLists(agonesVersion)
		ct.agonesVersion = agonesVersion
		if agonesVersion == "Dev" {
			ct.agonesVersion = Dev
			// Game server container cannot be created at DEV version due to go.mod only able to access
			// published Agones versions. Use N-1 for DEV.
			ct.fleetPath = createFleetFile(ReleaseVersion, countsAndLists)
		} else {
			ct.fleetPath = createFleetFile(agonesVersion, countsAndLists)
		}
		configTests = append(configTests, &ct)
	}

	return configTests
}

// containsCountsAndLists returns true if the agonesVersion >= 1.41.0 when the CountsAndLists
// feature entered Beta (on by default)
func containsCountsAndLists(agonesVersion string) bool {
	if agonesVersion == "Dev" {
		return true
	}
	r := regexp.MustCompile(`\d+\.\d+`)
	strVersion := r.FindString(agonesVersion)
	floatVersion, err := strconv.ParseFloat(strVersion, 64)
	if err != nil {
		log.Fatalf("Could not convert agonesVersion %s to float: %s", agonesVersion, err)
	}
	if floatVersion > 1.40 {
		return true
	}
	return false
}

// Finds the Kubernetes version of the Kubelet on the node that the current pod is running on.
// The Kubelet version is the same version as the node.
func findK8sVersion(ctx context.Context, kubeClient *kubernetes.Clientset) string {
	// Wait to get pod and node as these may take a while to start on a new Autopilot cluster.
	var pod *v1.Pod
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 7*time.Minute, true, func(ctx context.Context) (done bool, err error) {
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

		// TODO:
		// Create Watcher if the GameServer fails, (log.Fatalf), log info (current Agones version, Game
		//  server version, what Agones version is being installed) and fail this test and end job.
		createFleet(config.fleetPath)
	}
	// Wait for a bit of soak time after the last version of Agones has been installed before removing resources.
	time.Sleep(2 * time.Minute)
	// TODO: Delete fleets, uninstall Agones, delete namespace
	// kubectl get fleets --no-headers -o custom-columns=":metadata.name" | xargs kubectl delete fleets
	// helm uninstall agones -n agones-system
	// kubectl delete ns agones-system
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

// Creates a fleet yaml file from the mounted fleet.yaml template. The name of the new
// fleet yaml is based on the Agones version, i.e. fleet1440.yaml for Agones version 1.44.0
func createFleetFile(agonesVersion string, countsAndLists bool) string {
	fleetTmpl := gameServerTemplate{Registry: TestRegistry, AgonesVersion: agonesVersion, CountsAndLists: countsAndLists}

	fleetTemplate, err := template.ParseFiles("fleet.yaml")
	if err != nil {
		log.Fatal("Could not ParseFiles template fleet.yaml", err)
	}

	// Must use /tmp since the container is a non-root user.
	fleetPath := strings.ReplaceAll("/tmp/fleet"+agonesVersion, ".", "")
	fleetPath += ".yaml"
	// Check if the file already exists
	if _, err := os.Stat(fleetPath); err == nil {
		return fleetPath
	}
	fleetFile, err := os.Create(fleetPath)
	if err != nil {
		log.Fatal("Could not create file ", err)
	}

	err = fleetTemplate.Execute(fleetFile, fleetTmpl)
	if err != nil {
		if fErr := fleetFile.Close(); fErr != nil {
			log.Printf("Could not close fleet file %s, err: %s", fleetPath, fErr)
		}
		log.Fatal("Could not Execute template fleet.yaml ", err)
	}
	if err = fleetFile.Close(); err != nil {
		log.Printf("Could not close game server file %s, err: %s", fleetPath, err)
	}

	return fleetPath
}

// Installs a Fleet from a fleet.yaml file at the given Fleet path. The game server binary will be
// the sdk-client-test version in the fleet.yaml. The SDK version will be the same as the version of
// the Agones controller that created the game server. As the sdk-client-test shuts down after a run
// of tests the Fleet will create a new game server to replace it.
func createFleet(fleetPath string) {
	args := []string{"apply", "-f", fleetPath}

	backoff := wait.Backoff{
		Steps:    5,
		Duration: 1 * time.Second,
		Factor:   2.0,
		Jitter:   0.1,
	}

	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		_, err := runExecCommand(KubectlCmd, args...)

		switch {
		case err == nil:
			return true, nil
		default:
			return false, nil
		}
	})

	if err == nil {
		return
	}

	log.Fatalf("Could not create Fleet %s: %s", fleetPath, err)
}

// watchGameServerPods watches all pods for CrashLoopBackOff
func watchGameServerPods(kubeClient *kubernetes.Clientset, stopWatch chan struct{}) {
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, 5*time.Second)
	podInformer := kubeInformerFactory.Core().V1().Pods().Informer()

	// Filter by label agones.dev/role=gameserver

	_, err := podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			newPod := newObj.(*v1.Pod)
			for _, cs := range newPod.Status.ContainerStatuses {
				if cs.Name == "sdk-client-test" && cs.State.Waiting != nil {
					log.Printf("cs.State.Waiting.Reason: %s for pod: %s", cs.State.Waiting.Reason, newPod.Name)
				}
			}
		},
	})
	if err != nil {
		log.Fatal("Not able to create AddEventHandler", err)
	}

	go podInformer.Run(stopWatch)
	if !cache.WaitForCacheSync(stopWatch, podInformer.HasSynced) {
		log.Fatal("Timed out waiting for caches to sync")
	}
	<-stopWatch
}
