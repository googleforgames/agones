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

// Package main implements a program to remove the feature expiry version shortcodes in .md files.
package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	dirPath := "build/scripts"

	// Parse command-line arguments
	version := flag.String("version", "", "Expiry version to remove")
	flag.Parse()

	err := filepath.WalkDir(dirPath, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		// Read the content of the .md file
		file, err := os.Open(filePath)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Fatal(err)
			}
		}()

		scanner := bufio.NewScanner(file)
		modifiedContent := removeBlocks(scanner, *version)

		// Write the modified content back to the .md file
		outputFile, err := os.Create(filePath)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err := outputFile.Close(); err != nil {
				log.Fatal(err)
			}
		}()

		writer := bufio.NewWriter(outputFile)
		_, err = writer.WriteString(modifiedContent)
		if err != nil {
			log.Fatal(err)
		}
		err = writer.Flush()
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Processed file: %s\n", filePath)

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}

func removeBlocks(scanner *bufio.Scanner, version string) string {
	var sb strings.Builder
	expiryBlock := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line contains the starting of the expiryVersion shortcode with the specified version
		if strings.Contains(line, "{{% feature expiryVersion=\""+version+"\" %}}") {
			expiryBlock = true
			continue
		}

		// Check if the line contains the ending of the expiryVersion shortcode
		if strings.Contains(line, "{{% /feature %}}") && expiryBlock {
			expiryBlock = false
			continue
		}

		// Append the line if it is not within an expiryVersion block
		if !expiryBlock {
			_, err := sb.WriteString(line)
			if err != nil {
				log.Fatal(err)
			}
			_, err = sb.WriteString("\n")
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	return sb.String()
}