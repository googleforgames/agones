package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var releaseType string

func init() {
	flag.StringVar(&releaseType, "type", "", "Specify the release type ('before' or 'after')")
}

func main() {
	flag.Parse()

	if len(os.Args) < 3 {
		log.Fatalf("Please provide the release type ('before' or 'after') and the initial version as command-line arguments")
		return
	}

	releaseType := os.Args[1]
	initialVersion := os.Args[2]

	log.Printf("Release Type: %s", releaseType)
	log.Printf("Initial Version: %s", initialVersion)

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
		var err error
		if releaseType == "before" {
			err = UpdateValueInFileBeforeRelease(filename, initialVersion)
		} else if releaseType == "after" {
			err = UpdateVersionInFileAfterRelease(filename, initialVersion)
		} else {
			log.Fatalf("Invalid release type. Please specify 'before' or 'after'.")
		}

		if err != nil {
			log.Fatalf("Error updating file %s: %s\n", filename, err.Error())
		}
	}
}

func UpdateValueInFileBeforeRelease(filename string, valueToUpdate string) error {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	content := string(fileBytes)

	// Check the file extension and update values accordingly
	ext := filepath.Ext(filename)
	switch ext {
	case ".yaml", ".yml", ".nuspec", ".csproj":
		content = updateValuesBeforeRelease(content, valueToUpdate)
	case ".json":
		content = updateJSONValuesBeforeRelease(content, valueToUpdate)
	}

	err = ioutil.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}

	return nil
}

func updateValuesBeforeRelease(content string, valueToUpdate string) string {
	re := regexp.MustCompile(`(\d+\.\d+\.\d+)-dev`)
	return re.ReplaceAllString(content, "${1}")
}

func updateJSONValuesBeforeRelease(content string, valueToUpdate string) string {
	re := regexp.MustCompile(`"(\d+\.\d+\.\d+)-dev"`)
	return re.ReplaceAllString(content, `"$1"`)
}

func UpdateVersionInFileAfterRelease(filename string, initialVersion string) error {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	content := string(fileBytes)

	// Check the file extension and update version accordingly
	ext := filepath.Ext(filename)
	switch ext {
	case ".yaml", ".yml", ".nuspec", ".csproj":
		content = updateVersionAfterRelease(content, initialVersion)
	case ".json":
		content = updateJSONVersionAfterRelease(content, initialVersion)
	}

	err = ioutil.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}

	return nil
}

func updateVersionAfterRelease(content string, initialVersion string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(initialVersion))
	newVersion := incrementVersionAfterRelease(initialVersion)
	return re.ReplaceAllString(content, newVersion+"-dev")
}

func updateJSONVersionAfterRelease(content string, initialVersion string) string {
	re := regexp.MustCompile(`"` + regexp.QuoteMeta(initialVersion) + `"`)
	newVersion := incrementVersionAfterRelease(initialVersion) + "-dev"
	return re.ReplaceAllString(content, `"`+newVersion+`"`)
}

func incrementVersionAfterRelease(version string) string {
	segments := strings.Split(version, ".")
	lastButOneSegment, _ := strconv.Atoi(segments[len(segments)-2])
	segments[len(segments)-2] = strconv.Itoa(lastButOneSegment + 1)
	return strings.Join(segments, ".")
}
