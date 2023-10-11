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

// Package main implements a program that replaces ref with contents in the given json files
package main

import (
	"log"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 4 {
		log.Println("Usage: replaceRef <jsonFilePath> <ref> <contents>")
		return
	}

	jsonFilePath := os.Args[1]
	ref := os.Args[2]
	contents := os.Args[3]

	processJSON(jsonFilePath, ref, contents)
}

func processJSON(filename, ref, contents string) {
	log.Println("Processing JSON file:", filename)

	file, err := os.ReadFile(filename)
	if err != nil {
		log.Println("Error reading:", filename, err)
		return
	}

	searchPattern := "\"" + ref + "\": \"#/definitions/" + ref + "\""
	modifiedData := strings.ReplaceAll(string(file), searchPattern, contents)

	err = os.WriteFile(filename, []byte(modifiedData), 0o644)
	if err != nil {
		log.Println("Error writing:", filename, err)
		return
	}

	log.Println("JSON file processed successfully.")
}
