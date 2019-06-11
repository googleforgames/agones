// Copyright 2019 Google LLC All Rights Reserved.
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

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type testCase struct {
	Steps  []string `yaml:"steps,omitempty"`
	Result []string `yaml:"expected,omitempty"`
}

func runSidecar(language string) []bool {
	files, err := ioutil.ReadDir("./bin/" + language)
	if err != nil {
		log.Fatal(err)
	}
	var tc testCase
	testResults := make([]bool, 0)
	for _, f := range files {
		if f.Name() == "node_modules" || !f.IsDir() {
			continue
		}
		sidecar := "bin/sdk-server"
		cmdSdk := exec.Command(sidecar, "--local", "-f", "../../examples/gameserver.yaml",
			"--timeout", "15")
		sdkServer, err := cmdSdk.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}
		if err := cmdSdk.Start(); err != nil {
			log.Fatal(err)
		}
		//Wait for GRPC gateway to start
		time.Sleep(2 * time.Second)

		//cmd = exec.Command("cd", "./bin", "&&", "go", "build")

		yamlFile, err := ioutil.ReadFile("harness/" + f.Name() + ".yaml")
		if err != nil {
			log.Printf("yamlFile.Get err   #%v ", err)
		}

		if err := yaml.Unmarshal(yamlFile, &tc); err != nil {
			log.Fatal(err)
		}

		var cmdTest *exec.Cmd

		if language == "nodejs" {
			cmdTest = exec.Command("npm", "run", f.Name())
			cmdTest.Dir = "./bin/nodejs/"
		} else {
			cmdTest = exec.Command("./bin/golang/" + f.Name() + "/" + f.Name())
		}
		stderr, err := cmdTest.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}

		if err := cmdTest.Start(); err != nil {
			log.Fatal(err)
		}

		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			log.Printf("Line: %s\n", sc.Text())
		}

		sc = bufio.NewScanner(sdkServer)

		result := make(map[string]bool)
		for sc.Scan() {
			log.Printf("SDK Line : %s\n", sc.Text())

			for _, v := range tc.Result {
				if strings.Contains(sc.Text(), v) {
					result[v] = true
				}
			}
		}

		if err := cmdTest.Wait(); err != nil {
			log.Fatal(err)
		}

		if err := cmdSdk.Wait(); err != nil {
			log.Fatal(err)
		}
		all := true
		for _, v := range tc.Result {
			if _, ok := result[v]; !ok {
				all = false
				log.Printf("FAIL in %s file. Could not find expected: %s\n", f.Name(), v)
			}

		}
		testResults = append(testResults, all)
	}
	return testResults
}

// Code Generation function which would
// create a client code with test steps
// described by test files
func main() {
	lang := flag.String("sdk", "golang", "sdk language")
	testname := flag.String("test", "testReady", "test name")
	verify := flag.Bool("verify", false, "run sidecar and appropriate test")
	flag.Parse()
	log.Println(*lang)
	log.Println(*testname)
	// Enable line numbers in logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if *verify {
		testResults := runSidecar(*lang)
		log.Println("test Results: ", testResults)
		return
	}
	fileExtensions := map[string]string{"golang": "go", "nodejs": "js", "cpp": "cpp"}
	// Functions which contains code excerpts like init, ready, cleanup
	var functions struct {
		Functions map[string]string `yaml:"functions,omitempty"`
	}
	yamlFile, err := ioutil.ReadFile("sdk_client/" + *lang + ".yaml")
	if err != nil {
		log.Printf("yamlFile.Get err: %v ", err)
	}
	if err := yaml.Unmarshal(yamlFile, &functions); err != nil {
		log.Fatal(err)
	}

	for i := range functions.Functions {
		log.Printf("Function loaded '%s' \n", i)
	}

	testList := make([]string, 0)
	if *testname == "" {
		files, err := ioutil.ReadDir("./harness")
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			testList = append(testList, f.Name())
		}
	} else {
		testList = append(testList, *testname+".yaml")
	}

	var tc testCase
	for i, f := range testList {
		log.Println("Creating test ", i, " ", f)
		yamlFile, err = ioutil.ReadFile("harness/" + f)
		if err != nil {
			log.Printf("yamlFile.Get err: %v ", err)
		}

		if err := yaml.Unmarshal(yamlFile, &tc); err != nil {
			log.Fatal(err)
		}
		code := ""
		for _, v := range tc.Steps {
			if f, ok := functions.Functions[v]; ok {
				code += f
			} else {
				log.Printf("Could not find function declaration %s \n", v)
			}
		}
		test := f[:strings.IndexByte(f, '.')]
		path := "./bin/" + *lang + "/" + test
		err = os.MkdirAll(path, 0700)
		if err != nil {
			log.Printf("Create dir error: %v ", err)
		}
		err = ioutil.WriteFile(path+"/"+test+"."+fileExtensions[*lang], []byte(code), 0644)
		if err != nil {
			log.Printf("Write to file error: %v ", err)
		}
	}

	if *lang == "nodejs" {
		packagesBegin := `
{
"dependencies": {
"agones": "../../../../sdks/nodejs"
},
"scripts": {
			
		`
		packagesEnd := `
		
}
}	
`
		scripts := ""
		for i, f := range testList {
			test := f[:strings.IndexByte(f, '.')]
			scripts += fmt.Sprintf("\"%s\": \"node ./%s/%s.js\"", test, test, test)
			if i+1 < len(testList) {
				scripts += ",\n"
			}
		}
		packages := packagesBegin + scripts + packagesEnd
		err = ioutil.WriteFile("./bin/nodejs/package.json", []byte(packages), 0644)
		if err != nil {
			log.Printf("Write to file error: %v", err)
		}
	}
}
