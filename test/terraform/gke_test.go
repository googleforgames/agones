// Copyright 2020 Google LLC All Rights Reserved.
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

package terraform

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var project string
var tfPass string

func TestTerraformGKEInstallConfig(t *testing.T) {
	t.Logf("Using project %s\n", project)
	_, err := os.Stat(tfPass)
	assert.NoError(t, err)

	terraformOptions := &terraform.Options{
		TerraformDir: tfPass,
		Vars: map[string]interface{}{
			"project":     project,
			"name":        "terratest-cluster",
			"values_file": "",
		},
	}
	// run 'terraform apply' command
	// this would create GKE cluster,
	// deploy Agones of a release version and
	// create kubeconfig file
	terraform.InitAndApply(t, terraformOptions)

	defer func() {
		err := destroy(t, terraformOptions, tfPass)
		if err != nil {
			assert.FailNow(t, err.Error())
		}
	}()

	output := terraform.Output(t, terraformOptions, "host")
	assert.Contains(t, output, "https://")

}

// destroy runs terraform destroy on agones helm deployment.
// Second step is to run terraform destroy for all other resources including
// pregenerated kubeconfig file
func destroy(t *testing.T, options *terraform.Options, tfPass string) error {
	options.Targets = []string{"module.gke_helm.module.helm_agones.helm_release.agones"}
	terraform.Destroy(t, options)
	defer terraform.Destroy(t, options)

	namespaceName := "agones-system"
	options.Targets = []string{}

	// Setup the kubectl config and context. Here we choose to use the defaults, which is:
	// - HOME/.kube/config for the kubectl config file
	// - Current context of the kubectl config file
	kubectlOptions := k8s.NewKubectlOptions("", tfPass+"/kubeconfig", namespaceName)
	var i = 0
	// Wait for 120 seconds until all services are removed
	ticks := 12
	for i = 0; i < ticks; i++ {
		fmt.Println(i)
		svc, err := k8s.ListServicesE(t, kubectlOptions, metav1.ListOptions{LabelSelector: ""})
		assert.Nil(t, err)
		if len(svc) == 0 {
			break
		} else {
			for k, v := range svc {
				fmt.Println(k, v)
			}
			time.Sleep(10 * time.Second)
		}
	}
	if i == ticks {
		return fmt.Errorf("Failed waiting for services to be deleted")
	}
	return nil
}

func TestMain(m *testing.M) {
	pass := "./"
	projectFlag := flag.String("project", "agones", "name of the proejct")
	tfPassFlag := flag.String("tfpass", pass, "pass to terraform configs")
	flag.Parse()
	project = *projectFlag
	tfPass = *tfPassFlag
	m.Run()
}
