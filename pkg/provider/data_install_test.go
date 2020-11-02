/*
Copyright 2020 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataInstall_basic(t *testing.T) {
	resourceName := "data.flux_install.main"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				// Without required target_path set
				Config:      testAccDataInstallEmpty,
				ExpectError: regexp.MustCompile(`The argument "target_path" is required, but no definition was found\.`),
			},
			{
				// Check default values
				Config: testAccDataInstallBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "content"),
					resource.TestCheckResourceAttr(resourceName, "arch", "amd64"),
					resource.TestCheckResourceAttr(resourceName, "log_level", "info"),
					resource.TestCheckResourceAttr(resourceName, "namespace", "flux-system"),
					resource.TestCheckResourceAttr(resourceName, "network_policy", "true"),
					resource.TestCheckResourceAttr(resourceName, "path", "staging-cluster/flux-system/gotk-components.yaml"),
					resource.TestCheckResourceAttr(resourceName, "registry", "ghcr.io/fluxcd"),
					resource.TestCheckResourceAttr(resourceName, "target_path", "staging-cluster"),
					resource.TestCheckResourceAttr(resourceName, "version", "latest"),
					resource.TestCheckResourceAttr(resourceName, "watch_all_namespaces", "true"),
				),
			},
			// Ensure attribute value changes are propagated correctly into the state
			{
				Config: testAccDataInstallWithArg("arch", "arm64"),
				Check:  resource.TestCheckResourceAttr(resourceName, "arch", "arm64"),
			},
			{
				Config: testAccDataInstallWithArg("log_level", "debug"),
				Check:  resource.TestCheckResourceAttr(resourceName, "log_level", "debug"),
			},
			{
				Config: testAccDataInstallWithArg("namespace", "test-system"),
				Check:  resource.TestCheckResourceAttr(resourceName, "namespace", "test-system"),
			},
			{
				Config: testAccDataInstallWithArg("network_policy", "false"),
				Check:  resource.TestCheckResourceAttr(resourceName, "network_policy", "false"),
			},
			{
				Config: testAccDataInstallWithArg("version", "0.2.1"),
				Check:  resource.TestCheckResourceAttr(resourceName, "version", "0.2.1"),
			},
			{
				Config: testAccDataInstallWithArg("watch_all_namespaces", "false"),
				Check:  resource.TestCheckResourceAttr(resourceName, "watch_all_namespaces", "false"),
			},
		},
	})
}

const (
	testAccDataInstallEmpty = `data "flux_install" "main" {}`
	testAccDataInstallBasic = `
		data "flux_install" "main" {
			target_path = "staging-cluster"
		}
	`
)

func testAccDataInstallWithArg(attr string, value string) string {
	return fmt.Sprintf(`
		data "flux_install" "main" {
			target_path = "staging-cluster"
			%s = %q
		}
	`, attr, value)
}
