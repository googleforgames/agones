package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const examplesDir = "examples"

var excludedPatterns = [...]string{"*.md", "*.yaml", "OWNERS", ".gitignore"}

func dirIsExample(dirName string) bool {
	makefileName := fmt.Sprintf("%s/Makefile", dirName)
	if _, err := os.Stat(makefileName); err == nil {
		return true
	} else {
		return false
	}
}

func getAllExampleNames() []string {
	dirNames := make([]string, 0)

	baseDir := fmt.Sprintf("%s%s", rootDir, examplesDir)

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		log.Fatalf("Could not open directory: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		path := fmt.Sprintf("%s/%s", baseDir, entry.Name())
		if dirIsExample(path) {
			dirNames = append(dirNames, name)
		}
	}
	return dirNames
}

func getVersionFromMakefile(contents string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		line := scanner.Text()
		if lineContainsVersion(line) {
			return getVersionFromLine(line), nil
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Could not get version: %v", err)
	}

	return "", errors.New("no version could be found")
}

func lineContainsVersion(line string) bool {
	return strings.HasPrefix(line, "version :=")
}

func getVersionFromLine(line string) string {
	split := strings.SplitN(line, ":=", 2)
	if len(split) != 2 {
		log.Fatalf("Bad version line: %s", line)
	}
	trimmed := strings.TrimSpace(split[1])
	if trimmed == "" {
		log.Fatalf("Version can not be empty: %s", line)
	}
	return trimmed
}

func filenameIsIrrelevant(filename string, exampleNames []string) bool {
	if !filenameInExamples(filename, exampleNames) {
		return true
	}

	for _, excludedName := range excludedPatterns {
		matches, err := filepath.Match(excludedName, filename)
		if err != nil {
			log.Fatalf("Unknown error: %s", err)
		}

		if matches {
			return true
		}
	}
	return false
}

func filenameInExamples(filename string, exampleNames []string) bool {
	for _, exampleName := range exampleNames {
		path := fmt.Sprintf("%s/%s", examplesDir, exampleName)
		if strings.HasPrefix(filename, path) {
			return true
		}
	}
	return false
}
