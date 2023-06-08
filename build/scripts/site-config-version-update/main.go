// Copyright 2023 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main implements a program that replicates supported version of development to production.

package main

import (
	"log"
	"os"
	"strings"

	"github.com/pelletier/go-toml"
)

type Config struct {
	DevConfig struct {
		DevSupportedK8s                  []string `toml:"dev_supported_k8s"`
		DevK8sAPIVersion                 string   `toml:"dev_k8s_api_version"`
		DevGKEExampleClusterVersion      string   `toml:"dev_gke_example_cluster_version"`
		DevAKSExampleClusterVersion      string   `toml:"dev_aks_example_cluster_version"`
		DevEKSExampleClusterVersion      string   `toml:"dev_eks_example_cluster_version"`
		DevMinikubeExampleClusterVersion string   `toml:"dev_minikube_example_cluster_version"`
	} `toml:"params"`

	ProdConfig struct {
		SupportedK8s                  []string `toml:"supported_k8s"`
		K8sAPIVersion                 string   `toml:"k8s_api_version"`
		GKEExampleClusterVersion      string   `toml:"gke_example_cluster_version"`
		AKSExampleClusterVersion      string   `toml:"aks_example_cluster_version"`
		EKSExampleClusterVersion      string   `toml:"eks_example_cluster_version"`
		MinikubeExampleClusterVersion string   `toml:"minikube_example_cluster_version"`
	} `toml:"params"`
}

func main() {
	// Read the content of the config.toml file
	configFile := "site/config.toml"
	content, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatal("Read File: ", err)
	}

	// Unmarshal the TOML content into a Config struct
	var config Config
	err = toml.Unmarshal(content, &config)
	if err != nil {
		log.Fatal("Unmarshal: ", err)
	}

	// Copy values from dev to prod
	config.ProdConfig.SupportedK8s = config.DevConfig.DevSupportedK8s
	config.ProdConfig.K8sAPIVersion = config.DevConfig.DevK8sAPIVersion
	config.ProdConfig.GKEExampleClusterVersion = config.DevConfig.DevGKEExampleClusterVersion
	config.ProdConfig.AKSExampleClusterVersion = config.DevConfig.DevAKSExampleClusterVersion
	config.ProdConfig.EKSExampleClusterVersion = config.DevConfig.DevEKSExampleClusterVersion
	config.ProdConfig.MinikubeExampleClusterVersion = config.DevConfig.DevMinikubeExampleClusterVersion

	// Construct the updated TOML content
	var lines []string
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "supported_k8s") {
			line = "supported_k8s = " + tomlArray(config.ProdConfig.SupportedK8s)
		} else if strings.HasPrefix(line, "k8s_api_version") {
			line = "k8s_api_version = \"" + config.ProdConfig.K8sAPIVersion + "\""
		} else if strings.HasPrefix(line, "gke_example_cluster_version") {
			line = "gke_example_cluster_version = \"" + config.ProdConfig.GKEExampleClusterVersion + "\""
		} else if strings.HasPrefix(line, "aks_example_cluster_version") {
			line = "aks_example_cluster_version = \"" + config.ProdConfig.AKSExampleClusterVersion + "\""
		} else if strings.HasPrefix(line, "eks_example_cluster_version") {
			line = "eks_example_cluster_version = \"" + config.ProdConfig.EKSExampleClusterVersion + "\""
		} else if strings.HasPrefix(line, "minikube_example_cluster_version") {
			line = "minikube_example_cluster_version = \"" + config.ProdConfig.MinikubeExampleClusterVersion + "\""
		}
		lines = append(lines, line)
	}

	updatedContent := []byte(strings.Join(lines, "\n"))

	// Write the updated content back to the config.toml file
	err = os.WriteFile(configFile, updatedContent, 0644)
	if err != nil {
		log.Fatal("Write File: ", err)
	}

	log.Println("Values copied and saved successfully!")
}

// Helper function to convert an array of strings to a TOML array representation
func tomlArray(values []string) string {
	var builder strings.Builder
	builder.WriteString("[")
	for i, value := range values {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("\"")
		builder.WriteString(value)
		builder.WriteString("\"")
	}
	builder.WriteString("]")
	return builder.String()
}
