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
	TestRegistry = "us-docker.pkg.dev/agones-images/ci/sdk-client-test"
)

var (
	// Dev is the current development version of Agones
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

	validConfigs := configTestSetup(ctx, kubeClient)
	go watchGameServerPods(kubeClient, make(chan struct{}), make(map[string]podLog), len(validConfigs)*2)
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

type podLog struct {
	SdkVersion        string
	GameServerVersion string
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
		// TODO: create different valid config based off of available feature gates.
		// containsCountsAndLists will need to be updated to return true for when CountsAndLists=true.
		countsAndLists := containsCountsAndLists(agonesVersion)
		ct.agonesVersion = agonesVersion
		if agonesVersion == "Dev" {
			ct.agonesVersion = Dev
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

		go createGameServers(cancelCtx, config.gameServerPath)
		// Allow some soak time at the Agones version before next upgrade
		time.Sleep(1 * time.Minute)
	}
	cancel()
	// TODO: Replace sleep with wait for the existing healthy Game Servers finish naturally by reaching their shutdown phase.
	time.Sleep(30 * time.Second)
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
func createGameServerFile(agonesVersion string, countsAndLists bool) string {
	gsTmpl := gameServerTemplate{Registry: TestRegistry, AgonesVersion: agonesVersion, CountsAndLists: countsAndLists}

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
// be the same binary version as the game server file. The SDK version is always the same as the
// version of the Agones controller that created it. The Game Server shuts itself down after the
// tests have run as part of the `sdk-client-test` logic.
func createGameServers(ctx context.Context, gsPath string) {
	args := []string{"create", "-f", gsPath}
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			_, err := runExecCommand(KubectlCmd, args...)
			// TODO: Do not ignore error if unable to create due to something other than cluster scale up
			if err != nil {
				log.Printf("Could not create Gameserver %s: %s", gsPath, err)
			}
		}
	}
}

// watchGameServerPods watches all game server pods for CrashLoopBackOff. Errors if the number of
// CrashLoopBackOff backoff pods exceeds the number of acceptedFailures.
func watchGameServerPods(kubeClient *kubernetes.Clientset, stopCh chan struct{}, failedPods map[string]podLog, acceptedFailures int) {
	// Filter by label agones.dev/role=gameserver to only game server pods
	labelOptions := informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
		opts.LabelSelector = "agones.dev/role=gameserver"
	})
	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, 5*time.Second,
		informers.WithNamespace("default"), labelOptions)
	podInformer := kubeInformerFactory.Core().V1().Pods().Informer()

	_, err := podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			newPod := newObj.(*v1.Pod)
			for _, cs := range newPod.Status.ContainerStatuses {
				if cs.Name != "sdk-client-test" || cs.State.Waiting == nil || cs.State.Waiting.Reason != "CrashLoopBackOff" {
					continue
				}
				gsVersion := newPod.Labels["agonesVersion"]
				sdkVersion := newPod.Annotations["agones.dev/sdk-version"]
				log.Printf("%s for pod: %s with game server binary version %s, and SDK version %s", cs.State.Waiting.Reason, newPod.Name, gsVersion, sdkVersion)
				// Put failed pods into the map until it reaches capacity.
				failedPods[newPod.Name] = podLog{GameServerVersion: gsVersion, SdkVersion: sdkVersion}
				if len(failedPods) > acceptedFailures {
					log.Fatalf("Too many Game Server pods in CrashLoopBackOff: %v", failedPods)
				}
			}
		},
	})
	if err != nil {
		log.Fatal("Not able to create AddEventHandler", err)
	}

	go podInformer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, podInformer.HasSynced) {
		log.Fatal("Timed out waiting for caches to sync")
	}
}

// Deletes any remaining Game Servers, Uninstalls Agones, and Deletes agones-system namespace.
func cleanUpResources() {
	args := []string{"delete", "gs", "-l", "app=sdk-client-test"}
	_, err := runExecCommand(KubectlCmd, args...)
	if err != nil {
		log.Println("Could not delete game servers", err)
	}

	args = []string{"uninstall", "agones", "-n", "agones-system"}
	_, err = runExecCommand(HelmCmd, args...)
	if err != nil {
		log.Println("Could not Helm uninstall Agones", err)
	}

	// Apiservice v1.allocation.agones.dev, which is part of Service agones-system/agones-controller-service,
	// does not always get cleaned up on Helm uninstall, and needs to be deleted (if it exists) before
	// the agones-system namespace can be removed.
	// Ignore the error, because an "error" means Helm already uninstall the apiservice.
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
