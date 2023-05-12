package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide the initial version as a command-line argument")
		return
	}

	initialVersion := os.Args[1]
	files := []string{
		"install/helm/agones/Chart.yaml",
		"install/yaml/install.yaml",
		"install/helm/agones/values.yaml",
		"sdks/nodejs/package.json",
		"sdks/nodejs/package-lock.json",
		"sdks/unity/package.json",
		"sdks/csharp/sdk/AgonesSDK.nuspec",
		"sdks/csharp/sdk/csharp-sdk.csproj",
	}

	for _, filename := range files {
		err := UpdateVersionInFile(filename, initialVersion)
		if err != nil {
			fmt.Printf("Error updating file %s: %s\n", filename, err.Error())
		}
	}
}

func UpdateVersionInFile(filename string, initialVersion string) error {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	content := string(fileBytes)

	// Check the file extension and update version accordingly
	ext := filepath.Ext(filename)
	switch ext {
	case ".yaml", ".yml", ".nuspec", ".csproj":
		content = updateVersion(content, initialVersion)
	case ".json":
		content = updateJSONVersion(content, initialVersion)
	}

	err = ioutil.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}

	return nil
}

func updateVersion(content string, initialVersion string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(initialVersion))
	newVersion := incrementVersion(initialVersion)
	return re.ReplaceAllString(content, newVersion+"-dev")
}

func updateJSONVersion(content string, initialVersion string) string {
	re := regexp.MustCompile(`"` + regexp.QuoteMeta(initialVersion) + `"`)
	newVersion := incrementVersion(initialVersion) + "-dev"
	return re.ReplaceAllString(content, `"`+newVersion+`"`)
}

func incrementVersion(version string) string {
	segments := strings.Split(version, ".")
	lastButOneSegment, _ := strconv.Atoi(segments[len(segments)-2])
	segments[len(segments)-2] = strconv.Itoa(lastButOneSegment + 1)
	return strings.Join(segments, ".")
}
