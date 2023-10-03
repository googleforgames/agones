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

// Package main implements a program that updates the version of files in the sdks and install directories.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var releaseStage string
var version string

func init() {
	flag.StringVar(&releaseStage, "release-stage", "", "Specify the release stage ('before' or 'after')")
	flag.StringVar(&version, "version", "", "Specify the initial version")
}

func main() {
	flag.Parse()

	if releaseStage == "" || version == "" {
		log.Fatalf("Please provide the release stage ('before' or 'after') and the version as command-line arguments")
	}

	log.Printf("Release Stage: %s", releaseStage)
	log.Printf("Version: %s", version)

	files := []string{
		"build/Makefile",
		"install/helm/agones/Chart.yaml",
		"install/yaml/install.yaml",
		"install/helm/agones/values.yaml",
		"sdks/nodejs/package.json",
		"sdks/nodejs/package-lock.json",
		"sdks/unity/package.json",
		"sdks/csharp/sdk/AgonesSDK.nuspec",
		"sdks/csharp/sdk/csharp-sdk.csproj",
		"sdks/rust/Cargo.toml",
	}

	for _, filename := range files {
		// Print the directory path
		dir := filepath.Dir(filename)
		log.Printf("Directory: %s", dir)

		err := UpdateFile(filename, version)
		if err != nil {
			log.Fatalf("Error updating file %s: %s\n", filename, err.Error())
		}
	}
}

// UpdateFile updates the specified file to the current release version before and after the release process.
func UpdateFile(filename string, version string) error {
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	content := string(fileBytes)

	ext := filepath.Ext(filename)

	switch releaseStage {
	case "before":
		if ext == ".json" {
			re := regexp.MustCompile(`"(\d+\.\d+\.\d+)-dev"`)
			content = re.ReplaceAllString(content, `"$1"`)
		} else {
			re := regexp.MustCompile(`(\d+\.\d+\.\d+)-dev`)
			content = re.ReplaceAllString(content, "${1}")
		}
	case "after":
		if ext != ".json" {
			re := regexp.MustCompile(regexp.QuoteMeta(version))
			newVersion := incrementVersionAfterRelease(version)
			if filename == "build/Makefile" {
				content = re.ReplaceAllString(content, newVersion)
			} else {
				content = re.ReplaceAllString(content, newVersion+"-dev")
			}
		} else {
			re := regexp.MustCompile(`"` + regexp.QuoteMeta(version) + `"`)
			newVersion := incrementVersionAfterRelease(version) + "-dev"
			content = re.ReplaceAllString(content, `"`+newVersion+`"`)
		}
	default:
		log.Fatalf("Invalid release stage. Please specify 'before' or 'after'.")
	}

	err = os.WriteFile(filename, []byte(content), 0o644)
	if err != nil {
		return err
	}

	return nil
}

func incrementVersionAfterRelease(version string) string {
	segments := strings.Split(version, ".")
	if len(segments) < 3 {
		log.Fatalf("Invalid version format: %s\n", version)
	}

	lastButOneSegment, err := strconv.Atoi(segments[len(segments)-2])
	if err != nil {
		log.Fatalf("Error converting version segment to integer: %s\n", err.Error())
	}
	segments[len(segments)-2] = strconv.Itoa(lastButOneSegment + 1)
	return strings.Join(segments, ".")
}
