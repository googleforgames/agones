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

// Package main implements a program that updates the version of Agones package in sdks/rust/Cargo.toml file.
package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	configFile := "sdks/rust/Cargo.toml"
	content, err := os.ReadFile(configFile)
	if err != nil {
		log.Println("Read File: ", err)
	}

	var lines []string
	lines = append(lines, strings.Split(string(content), "\n")...)

	updatedLines := updateReleaseValues(lines)

	err = writeLinesToFile(configFile, updatedLines)
	if err != nil {
		log.Println("Write File: ", err)
	}
}

func updateReleaseValues(lines []string) []string {
	var updatedLines []string

	shouldUpdateVersion := false

	for _, line := range lines {
		updatedLine := line

		if strings.Contains(line, `name = "agones"`) {
			shouldUpdateVersion = true
		}

		// Update the version if the flag is set
		if shouldUpdateVersion && strings.HasPrefix(line, "version") {
			re := regexp.MustCompile(`"[^"]+"`)
			match := re.FindString(line)
			if match != "" {
				version := strings.Trim(match, `"`)

				segments := strings.Split(version, ".")

				if len(segments) == 3 {
					secondSegment, _ := strconv.Atoi(segments[1])
					secondSegment++
					segments[1] = strconv.Itoa(secondSegment)
					updatedVersion := strings.Join(segments, ".")
					updatedLine = strings.Replace(line, version, updatedVersion, 1)
				}
			}

			// Reset the flag after updating the version
			shouldUpdateVersion = false
		}

		updatedLines = append(updatedLines, updatedLine)
	}

	return updatedLines
}

func writeLinesToFile(filePath string, lines []string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Println(err)
		}
	}()

	writer := bufio.NewWriter(file)

	for i, line := range lines {
		if i < len(lines)-1 {
			_, err := writer.WriteString(line + "\n")
			if err != nil {
				return err
			}
		} else {
			_, err := writer.WriteString(line)
			if err != nil {
				return err
			}
		}
	}

	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}
