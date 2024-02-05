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
// limitations under the License.

// Package main implements a program to increment the new tag for the given examples image. Run this script using `make bump-image IMAGENAME=<imageName> VERSION=<version>`
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	imageName       string
	version         string
	versionPattern  *regexp.Regexp
	targetedFolders = map[string]bool{
		"build":    true,
		"examples": true,
		"install":  true,
		"pkg":      true,
		"site":     true,
		"test":     true,
	}
)

func init() {
	flag.StringVar(&imageName, "imageName", "", "Image name to update")
	flag.StringVar(&version, "version", "", "Version to update to")
}

func main() {
	flag.Parse()

	if imageName == "" || version == "" {
		log.Fatal("Provide both an image name and a version using the flags.")
	}

	versionPatternString := imageName + `:(\d+)\.(\d+)`
	versionPattern = regexp.MustCompile(versionPatternString)
	newVersion := incrementVersion(version)

	baseDirectory := "."
	for folder := range targetedFolders {
		directory := filepath.Join(baseDirectory, folder)

		err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) != ".md" {
				err = updateFileVersion(path, newVersion)
				if err != nil {
					log.Printf("Error updating file %s: %v", path, err)
				}
			}
			return nil
		})

		if err != nil {
			log.Fatalf("Error processing directory %s: %v", directory, err)
		}
	}
}

func incrementVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		log.Fatalf("Invalid version format: %s", version)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Fatalf("Invalid version number: %v", err)
	}

	minor++
	return fmt.Sprintf("%s.%d", parts[0], minor)
}

func updateFileVersion(filePath, newVersion string) error {
	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := versionPattern.ReplaceAllString(string(input), imageName+":"+newVersion)

	return os.WriteFile(filePath, []byte(content), 0o644)
}
