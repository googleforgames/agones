package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide the value to update as a command-line argument")
		return
	}

	valueToUpdate := os.Args[1]

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
		err := UpdateValueInFile(filename, valueToUpdate)
		if err != nil {
			fmt.Printf("Error updating file %s: %s\n", filename, err.Error())
		}
	}
}

func UpdateValueInFile(filename string, valueToUpdate string) error {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	content := string(fileBytes)

	// Check the file extension and update values accordingly
	ext := filepath.Ext(filename)
	switch ext {
	case ".yaml", ".yml", ".nuspec", ".csproj":
		content = updateValues(content, valueToUpdate)
	case ".json":
		content = updateJSONValues(content, valueToUpdate)
	}

	err = ioutil.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}

	return nil
}

func updateValues(content string, valueToUpdate string) string {
	re := regexp.MustCompile(`(\d+\.\d+\.\d+)-dev`)
	return re.ReplaceAllString(content, "${1}")
}

func updateJSONValues(content string, valueToUpdate string) string {
	re := regexp.MustCompile(`"(\d+\.\d+\.\d+)-dev"`)
	return re.ReplaceAllString(content, `"$1"`)
}
