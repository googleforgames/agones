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

// Package main implements a program to convert from export-openapi.sh to Go script.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	tmpDir := "../tmp"

	if _, err := os.Stat(tmpDir); err == nil {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			log.Fatal(err)
		}
	}

	err := os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	// Start the kubectl proxy.
	cmd := exec.Command("kubectl", "proxy")
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	// Make an HTTP request to fetch the OpenAPI JSON.
	resp, err := http.Get("http://127.0.0.1:8001/openapi/v2")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Create a file to store the OpenAPI JSON in the `tmp` directory.
	openapiFile := filepath.Join(tmpDir, "openapi.json")
	file, err := os.Create(openapiFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Copy the HTTP response body to the file.
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Remove specified fields from the OpenAPI JSON.
	err = removeFields(openapiFile)
	if err != nil {
		log.Fatal(err)
	}

	// Read the modified OpenAPI JSON file.
	file, err = os.Open(openapiFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var openAPI map[string]interface{}
	err = json.NewDecoder(file).Decode(&openAPI)
	if err != nil {
		log.Fatal(err)
	}

	doExpand("io.k8s.api.core.v1.PodTemplateSpec", tmpDir, openAPI)

	objectMetaPath := filepath.Join(tmpDir, "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta.json")
	if err := makeCreationTimestampNullable(objectMetaPath); err != nil {
		log.Fatal(err)
	}

	intOrStringPath := filepath.Join(tmpDir, "io.k8s.apimachinery.pkg.util.intstr.IntOrString.json")
	if err := modifyIntOrString(intOrStringPath); err != nil {
		log.Fatal(err)
	}

	podSpecPath := filepath.Join(tmpDir, "io.k8s.api.core.v1.PodSpec.json")
	if err := escapeDoubleBackslashes(podSpecPath); err != nil {
		log.Fatal(err)
	}

}

func removeFields(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var openAPI map[string]interface{}
	err = json.NewDecoder(file).Decode(&openAPI)
	if err != nil {
		return err
	}

	keysToRemove := []string{
		"x-kubernetes-patch-strategy",
		"x-kubernetes-patch-merge-key",
		"x-kubernetes-list-type", // ******DOUBLE CHECK - this key is present in two places on openapi.json file
		"x-kubernetes-group-version-kind",
		"x-kubernetes-list-map-keys",
		"x-kubernetes-unions",
	}

	removeKeysRecursively(openAPI, keysToRemove)

	file, err = os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(openAPI)
	if err != nil {
		return err
	}

	return nil
}

func removeKeysRecursively(obj interface{}, keysToRemove []string) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for _, key := range keysToRemove {
			delete(v, key)
		}
		for _, value := range v {
			removeKeysRecursively(value, keysToRemove)
		}
	case []interface{}:
		for _, item := range v {
			removeKeysRecursively(item, keysToRemove)
		}
	}
}

func doExpand(key, tmpDir string, openapi map[string]interface{}) {
	log.Println("Processing", key)

	// Extract and save the definitions
	definitions := openapi["definitions"].(map[string]interface{})
	section, exists := definitions[key]
	if !exists {
		log.Printf("Key %s not found in definitions\n", key)
		return
	}

	sectionJSON, err := json.MarshalIndent(section, "", "  ")
	if err != nil {
		log.Println("Error marshaling section:", err)
		return
	}

	err = os.WriteFile(filepath.Join(tmpDir, key+".json"), sectionJSON, 0o644)
	if err != nil {
		log.Println("Error writing to file:", err)
		return
	}

	// Recursively search for $ref values
	var children []string
	err = searchForRefs(section, &children)
	if err != nil {
		log.Println("Error searching for references:", err)
		return
	}

	for _, child := range children {
		doExpand(child, tmpDir, openapi)
	}
}

func searchForRefs(obj interface{}, refs *[]string) error {
	switch v := obj.(type) {
	case map[string]interface{}:
		if ref, ok := v["$ref"]; ok {
			refStr := ref.(string)
			refStr = strings.TrimPrefix(refStr, "#/definitions/")
			*refs = append(*refs, refStr)
		}

		for _, value := range v {
			err := searchForRefs(value, refs)
			if err != nil {
				return err
			}
		}
	case []interface{}:
		for _, item := range v {
			err := searchForRefs(item, refs)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func makeCreationTimestampNullable(filePath string) error {
	var obj map[string]interface{}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	if properties, ok := obj["properties"].(map[string]interface{}); ok {
		if timestamp, ok := properties["creationTimestamp"].(map[string]interface{}); ok {
			timestamp["nullable"] = true
		}
	}

	modifiedData, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, modifiedData, 0o644)
}

func modifyIntOrString(filePath string) error {
	var obj map[string]interface{}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	obj["x-kubernetes-int-or-string"] = true
	delete(obj, "type")

	modifiedData, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, modifiedData, 0o644)
}

// DOUBLE CHECK
func escapeDoubleBackslashes(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	modifiedData := strings.ReplaceAll(string(data), "\\\\", "\\\\\\\\")
	return os.WriteFile(filePath, []byte(modifiedData), 0o644)
}
