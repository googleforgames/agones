// Copyright 2023 Google LLC All Rights Reserved.
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
// limitations under the License

// evictpod.go --pod <pod> --namespace <namespae> initiates a pod eviction
// using the k8s eviction API.
package main

import (
	"context"
	"flag"
	"log"
	"path/filepath"

	policy "k8s.io/api/policy/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Borrowed from https://stackoverflow.com/questions/62803041/how-to-evict-or-delete-pods-from-kubernetes-using-golang-client
func evictPod(client *kubernetes.Clientset, name, namespace string) error {
	return client.PolicyV1beta1().Evictions(namespace).Evict(context.TODO(), &policy.Eviction{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		}})
}

func main() {
	kubeconfig := flag.String("kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	namespace := flag.String("namespace", "default", "Namespace (required)")
	pod := flag.String("pod", "", "Pod name (required)")
	flag.Parse()

	if *pod == "" || *namespace == "" {
		log.Fatal("--pod and --namespace must be non-empty")
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatalf("Could not create in cluster config: %v", err)
	}

	var kubeClient *kubernetes.Clientset
	kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Could not create the kubernetes clientset: %v", err)
	}

	err = evictPod(kubeClient, *pod, *namespace)
	if err != nil {
		log.Fatalf("Pod eviction failed: %v", err)
	}
}
