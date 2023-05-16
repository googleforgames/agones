package main

import (
	"flag"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var releaseStage string

func init() {
	flag.StringVar(&releaseStage, "release-stage", "", "Specify the release stage ('before' or 'after')")
}

func main() {
	flag.Parse()

	if len(flag.Args()) < 1 {
		log.Fatalf("Please provide the release stage ('before' or 'after') and the initial version as command-line arguments")
	}

	initialVersion := flag.Arg(0)

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
		err := UpdateFile(filename, initialVersion)
		if err != nil {
			log.Fatalf("Error updating file %s: %s\n", filename, err.Error())
		}
	}
}

func UpdateFile(filename string, initialVersion string) error {
	fileBytes, err := ioutil.ReadFile(filename)
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
		if ext == ".json" {
			re := regexp.MustCompile(`"` + regexp.QuoteMeta(initialVersion) + `"`)
			newVersion := incrementVersionAfterRelease(initialVersion) + "-dev"
			content = re.ReplaceAllString(content, `"`+newVersion+`"`)
		} else {
			re := regexp.MustCompile(regexp.QuoteMeta(initialVersion))
			newVersion := incrementVersionAfterRelease(initialVersion)
			content = re.ReplaceAllString(content, newVersion+"-dev")
		}
	default:
		log.Fatalf("Invalid release stage. Please specify 'before' or 'after'.")
	}

	err = ioutil.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}

	return nil
}

func incrementVersionAfterRelease(version string) string {
	segments := strings.Split(version, ".")
	lastButOneSegment, _ := strconv.Atoi(segments[len(segments)-2])
	segments[len(segments)-2] = strconv.Itoa(lastButOneSegment + 1)
	return strings.Join(segments, ".")
}
