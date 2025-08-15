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

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
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
	// Timeout sets the amount of time to wait for resources to become ready. Should be more than the
	// time for an Autopilot cluster to scale up.
	Timeout = 10 * time.Minute
	// HelmChart is the helm chart for the public Agones releases
	HelmChart = "agones/agones"
	// TestChart is the registry for Agones Helm chart development builds
	TestChart = "./install/helm"
	// AgonesRegistry is the public registry for Agones releases
	AgonesRegistry = "us-docker.pkg.dev/agones-images/release"
	// TestRegistry is the registry for Agones development builds
	TestRegistry = "us-docker.pkg.dev/agones-images/ci"
	// ContainerRegistry is the registry for upgrade test container files
	ContainerRegistry = "us-docker.pkg.dev/agones-images/ci/sdk-client-test"
)

var (
	// DevVersion is the current development version of Agones
	DevVersion = os.Getenv("DevVersion")
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

	agonesClient, err := versioned.NewForConfig(cfg)
	if err != nil {
		log.Fatal("Could not create the agones api clientset")
	}

	validConfigs := configTestSetup(ctx, kubeClient)
	go watchGameServers(agonesClient, len(validConfigs)*2)
	go watchGameServerEvents(kubeClient)
	addAgonesRepo()
	runConfigWalker(ctx, validConfigs)
	cleanUpResources()
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

// CountsAndLists can be removed from the template once CountsAndLists is GA in all tested versions
type gameServerTemplate struct {
	AgonesVersion  string
	Registry       string
	CountsAndLists bool
}

type gsLog struct {
	SdkVersion        string
	GameServerVersion string
	GameServerState   string
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
		log.Fatal("Could not Unmarshal ", err)
	}

	// Find valid Agones versions and feature gates for the current version of Kubernetes.
	configTests := []*configTest{}
	for _, agonesVersion := range versionMap.K8sToAgonesVersions[k8sVersion] {
		ct := configTest{}
		// TODO: create different valid config based off of available feature gates.
		// containsCountsAndLists will need to be updated to return true for when CountsAndLists=true.
		countsAndLists := containsCountsAndLists(agonesVersion)
		ct.agonesVersion = agonesVersion
		if agonesVersion == "Dev" {
			ct.agonesVersion = DevVersion
			// Game server container cannot be created at DEV version due to go.mod only able to access
			// published Agones versions. Use N-1 for DEV.
			ct.gameServerPath = createGameServerFile(ReleaseVersion, countsAndLists)
		} else {
			ct.gameServerPath = createGameServerFile(agonesVersion, countsAndLists)
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
	cancelCtx, cancel := context.WithCancel(ctx)

	for _, config := range validConfigs {
		registry := AgonesRegistry
		chart := HelmChart
		if config.agonesVersion == DevVersion {
			registry = TestRegistry
			chart = TestChart
		}
		err := installAgonesRelease(config.agonesVersion, registry, config.featureGates, ImagePullPolicy,
			SidecarPullPolicy, LogLevel, chart)
		if err != nil {
			log.Fatalf("installAgonesRelease err: %s", err)
		}

		// Wait for the helm release to install. Waits the same amount of time as the Helm timeout.
		var helmStatus string
		err = wait.PollUntilContextTimeout(ctx, 10*time.Second, Timeout, true, func(_ context.Context) (done bool, err error) {
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

		gsReady := make(chan bool)
		go createGameServers(cancelCtx, config.gameServerPath, gsReady)
		// Wait for the first game server pod created to become ready
		<-gsReady
		close(gsReady)
		// Allow some soak time at the Agones version before next upgrade
		time.Sleep(1 * time.Minute)
	}
	cancel()
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

	// Remove the commit sha from the DevVersion i.e. from 1.46.0-dev-7168dd3 to 1.46.0-dev or 1.46.0
	// for the case of this test running during a new Agones version release PR.
	agonesRelease := ""
	if agonesVersion == DevVersion {
		r := regexp.MustCompile(`1\.\d+\.\d+-dev`)
		agonesVersion = r.FindString(DevVersion)
		r = regexp.MustCompile(`1\.\d+\.\d`)
		agonesRelease = r.FindString(agonesVersion)
	}

	for _, status := range helmStatus {
		if (status.AppVersion == agonesVersion) || (status.AppVersion == agonesRelease) {
			return status.Status
		}
	}
	return ""
}

// Creates a gameserver yaml file from the mounted gameserver.yaml template. The name of the new
// gameserver yaml is based on the Agones version, i.e. gs1440.yaml for Agones version 1.44.0
// Note: This does not validate the created file.
func createGameServerFile(agonesVersion string, countsAndLists bool) string {
	gsTmpl := gameServerTemplate{Registry: ContainerRegistry, AgonesVersion: agonesVersion, CountsAndLists: countsAndLists}

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

// Create a game server every five seconds until the context is cancelled. The game server container
// is the same binary version as the game server file. The SDK version is always the same as the
// version of the Agones controller that created it. The Game Server shuts itself down after the
// tests have run as part of the `sdk-client-test` logic.
func createGameServers(ctx context.Context, gsPath string, gsReady chan bool) {
	args := []string{"create", "-f", gsPath}
	checkFirstGameServerReady(ctx, gsReady, args...)

	ticker := time.NewTicker(5 * time.Second)
	retries := 8
	retry := 0

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			_, err := runExecCommand(KubectlCmd, args...)
			// Ignore failures for ~45s at at time to account for the brief (~30s) during which the
			// controller service is unavailable during upgrade.
			if err != nil {
				if retry > retries {
					log.Fatalf("Could not create Gameserver %s: %s. Too many successive errors.", gsPath, err)
				}
				log.Printf("Could not create Gameserver %s: %s. Retries left: %d.", gsPath, err, retries-retry)
				retry++
			} else {
				retry = 0
			}
		}
	}
}

// checkFirstGameServerReady waits for the Game Server Pod to be running. This may take several
// minutes in Autopilot.
func checkFirstGameServerReady(ctx context.Context, gsReady chan bool, args ...string) {
	// Sample output: gameserver.agones.dev/sdk-client-test-5zjdn created
	output, err := runExecCommand(KubectlCmd, args...)
	if err != nil {
		log.Fatalf("Could not create Gameserver: %s", err)
	}
	r := regexp.MustCompile(`sdk-client-test-\S+`)
	gsName := r.FindString(string(output))
	// Game Server has too many states, so using the pod instead as there are only two healthy states.
	// Includes the gs name to make output logs easier to read.
	getPodStatus := []string{"get", "pod", gsName, "-o=custom-columns=:.status.phase,:.metadata.name", "--no-headers"}

	// Pod is created after Game Server, wait briefly before erroring out on unable to get pod.
	retries := 0
	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, Timeout, true, func(_ context.Context) (done bool, err error) {
		out, err := runExecCommand(KubectlCmd, getPodStatus...)
		if err != nil && retries > 2 {
			log.Fatalf("Could not get Gameserver %s state: %s", gsName, err)
		}
		if err != nil {
			retries++
			return false, nil
		}
		// Sample output: Running   sdk-client-test-bbvx9
		podStatus := strings.Split(string(out), " ")
		if podStatus[0] == "Running" || podStatus[0] == "Succeeded" {
			gsReady <- true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		log.Fatalf("PollUntilContextTimeout timed out while wait for first gameserver %s to be Ready", gsName)
	}
}

// watchGameServers watches all game servers for errors. Errors if the number of failed game servers
// exceeds the number of acceptedFailures.
func watchGameServers(agonesClient *versioned.Clientset, acceptedFailures int) {
	stopCh := make(chan struct{})
	failedGs := make(map[string]gsLog)

	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, 5*time.Second)
	gsInformer := agonesInformerFactory.Agones().V1().GameServers().Informer()

	_, err := gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			newGs := newObj.(*agonesv1.GameServer)
			if newGs.Status.State == "Error" || newGs.Status.State == "Unhealthy" {
				gsVersion := newGs.Labels["agonesVersion"]
				sdkVersion := newGs.Annotations["agones.dev/sdk-version"]
				log.Printf("Game server %s with binary version %s, and SDK version %s in %s state\n",
					newGs.Name, gsVersion, sdkVersion, newGs.Status.State)

				// Put failed game servers into the map until it reaches capacity.
				failedGs[newGs.Name] = gsLog{GameServerVersion: gsVersion, SdkVersion: sdkVersion,
					GameServerState: string(newGs.Status.State)}
				if len(failedGs) > acceptedFailures {
					log.Fatalf("Too many Game Servers in Error or Unhealthy states: %v", failedGs)
				}
			}
		},
	})
	if err != nil {
		log.Fatal("Not able to create AddEventHandler", err)
	}

	go gsInformer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, gsInformer.HasSynced) {
		log.Fatal("Timed out waiting for game server informer cache to sync")
	}
}

// watchGameServerEvents watches all events on `sdk-client-test` containers for BackOff errors. The
// purpose is to catch ImagePullBackOff errors.
func watchGameServerEvents(kubeClient *kubernetes.Clientset) {
	stopCh := make(chan struct{})

	// Filter by Game Server `sdk-client-test` containers
	containerName := "sdk-client-test"
	containerPath := "spec.containers{sdk-client-test}"
	fieldSelector := fields.OneTermEqualSelector("involvedObject.fieldPath", containerPath).String()
	// First delete previous `sdk-client-test` events, otherwise there will be events from previous runs.
	_, err := runExecCommand(KubectlCmd, []string{"delete", "events", "--field-selector", fieldSelector}...)
	if err != nil {
		log.Fatal("Could not delete `sdk-client-test` events", err)
	}

	eventOptions := informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
		opts.FieldSelector = fieldSelector
	})
	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, 5*time.Second,
		informers.WithNamespace("default"), eventOptions)
	eventInformer := kubeInformerFactory.Core().V1().Events().Informer()

	_, err = eventInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newEvent := obj.(*v1.Event)
			gsPodName := newEvent.InvolvedObject.Name
			if newEvent.Reason == "Failed" {
				log.Fatalf("%s on %s %s has failed. Latest event: message %s", containerName, newEvent.Kind,
					gsPodName, newEvent.Message)
			}
		},
	})
	if err != nil {
		log.Fatal("Not able to create AddEventHandler", err)
	}

	go eventInformer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, eventInformer.HasSynced) {
		log.Fatal("Timed out waiting for eventInformer cache to sync")
	}
}

// Deletes any remaining Game Servers, Uninstalls Agones, and Deletes agones-system namespace.
func cleanUpResources() {
	args := []string{"delete", "gs", "-l", "app=sdk-client-test"}
	_, err := runExecCommand(KubectlCmd, args...)
	if err != nil {
		log.Println("Could not delete game servers", err)
	}

	args = []string{"uninstall", "agones", "-n", "agones-system", "--wait", "--timeout", "10m", "--debug"}
	_, err = runExecCommand(HelmCmd, args...)
	if err != nil {
		log.Println("Could not Helm uninstall Agones", err)
	}

	// Apiservice v1.allocation.agones.dev, which is part of Service agones-system/agones-controller-service,
	// does not always get cleaned up on Helm uninstall, and needs to be deleted (if it exists) before
	// the agones-system namespace can be removed.
	// Ignore the error, because an "error" means Helm already uninstalled the apiservice.
	args = []string{"delete", "apiservice", "v1.allocation.agones.dev"}
	out, err := runExecCommand(KubectlCmd, args...)
	if err == nil {
		fmt.Println(string(out))
	}

	args = []string{"delete", "ns", "agones-system"}
	_, err = runExecCommand(KubectlCmd, args...)
	if err != nil {
		log.Println("Could not delete agones-system namespace", err)
	}
}
