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

// Package main implements a program to remove data-proofer-ignore attribute from previous release blog
package main

import (
	"flag"
	"log"
	"os"
	"strings"
)

func main() {
	// Define a flag to accept the file name
	fileName := flag.String("file", "", "Path to the file")
	flag.Parse()

	if *fileName == "" {
		log.Fatal("Please provide the file name using the -file flag")
	}

	filePath := "site/content/en/blog/releases/" + *fileName

	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Read the file content
	stat, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	content := make([]byte, stat.Size())
	_, err = file.Read(content)
	if err != nil {
		log.Fatal(err)
	}

	contentStr := string(content)

	// Remove the "data-proofer-ignore" word from the content
	modifiedContent := strings.Replace(contentStr, "data-proofer-ignore", "", -1)

	// Truncate the file before writing the modified content
	err = file.Truncate(0)
	if err != nil {
		log.Fatal(err)
	}

	// Move the file offset to the beginning
	_, err = file.Seek(0, 0)
	if err != nil {
		log.Fatal(err)
	}

	// Write the modified content back to the file
	_, err = file.WriteString(modifiedContent)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("File successfully modified:", filePath)
}
