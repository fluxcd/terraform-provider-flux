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
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDataInstall_basic(t *testing.T) {
	resourceName := "data.flux_install.main"
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Without required target_path set
				Config:      testAccDataInstallEmpty,
				ExpectError: regexp.MustCompile(`The argument "target_path" is required, but no definition was found\.`),
			},
			{
				// With invalid log level
				Config:      testAccDataInstallLogLevel,
				ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
			},
			{
				// Check default values
				Config: testAccDataInstallBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "content"),
					resource.TestCheckResourceAttr(resourceName, "components.#", "4"),
					resource.TestCheckResourceAttr(resourceName, "log_level", "info"),
					resource.TestCheckResourceAttr(resourceName, "namespace", "flux-system"),
					resource.TestCheckResourceAttr(resourceName, "cluster_domain", "cluster.local"),
					resource.TestCheckResourceAttr(resourceName, "network_policy", "true"),
					resource.TestCheckResourceAttr(resourceName, "path", "staging-cluster/flux-system/gotk-components.yaml"),
					resource.TestCheckResourceAttr(resourceName, "registry", "ghcr.io/fluxcd"),
					resource.TestCheckResourceAttr(resourceName, "target_path", "staging-cluster"),
					resource.TestCheckResourceAttr(resourceName, "watch_all_namespaces", "true"),
					resource.TestCheckResourceAttr(resourceName, "baseurl", "https://github.com/fluxcd/flux2/releases"),
				),
			},
			// Ensure attribute value changes are propagated correctly into the state
			{
				Config: testAccDataInstallWithArg("log_level", "debug"),
				Check:  resource.TestCheckResourceAttr(resourceName, "log_level", "debug"),
			},
			{
				Config: testAccDataInstallWithArg("namespace", "test-system"),
				Check:  resource.TestCheckResourceAttr(resourceName, "namespace", "test-system"),
			},
			{
				Config: testAccDataInstallWithArg("cluster_domain", "k8s.local"),
				Check:  resource.TestCheckResourceAttr(resourceName, "cluster_domain", "k8s.local"),
			},
			{
				Config: testAccDataInstallWithListArg("components", []string{"source-controller", "kustomize-controller"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "components.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "components.0", "kustomize-controller"),
					resource.TestCheckResourceAttr(resourceName, "components.1", "source-controller"),
				),
			},
			{
				Config: testAccDataInstallWithListArg("components_extra", []string{"image-automation-controller"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "components_extra.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "components_extra.0", "image-automation-controller"),
				),
			},
			{
				Config: testAccDataInstallWithArg("network_policy", "false"),
				Check:  resource.TestCheckResourceAttr(resourceName, "network_policy", "false"),
			},
			{
				Config: testAccDataInstallWithArg("watch_all_namespaces", "false"),
				Check:  resource.TestCheckResourceAttr(resourceName, "watch_all_namespaces", "false"),
			},
			{
				Config:      testAccDataInstallWithArg("version", "foo"),
				ExpectError: regexp.MustCompile("must either be latest or start with 'v'"),
			},
			{
				Config:      testAccDataInstallWithArg("baseurl", "http://www.example.org"),
				ExpectError: regexp.MustCompile("failed to download manifests.tar.gz"),
			},
			{
				Config: testAccDataInstallWithArg("version", "v0.5.3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "version", "v0.5.3"),
					func(s *terraform.State) error {
						install, ok := s.RootModule().Resources["data.flux_install.main"]
						if !ok {
							return fmt.Errorf("did not find expected flux_install datasource")
						}

						content := install.Primary.Attributes["content"]
						images := []string{
							"source-controller:v0.5.4", "kustomize-controller:v0.5.0", "notification-controller:v0.5.0", "helm-controller:v0.4.3",
						}
						for _, image := range images {
							imageKey := fmt.Sprintf("ghcr.io/fluxcd/%s", image)
							if !strings.Contains(content, imageKey) {
								return fmt.Errorf("expected %q to be present in the manifest content", imageKey)
							}
						}

						return nil
					},
				),
			},
			{
				Config: testAccDataInstallTolerationKeys,
				Check:  resource.TestCheckResourceAttr(resourceName, "toleration_keys.#", "2"),
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
	testAccDataInstallLogLevel = `
		data "flux_install" "main" {
			target_path = "staging-cluster"
			log_level   = "warn"
		}
	`
	testAccDataInstallTolerationKeys = `
		data "flux_install" "main" {
			target_path = "staging-cluster"
			toleration_keys = ["foo", "bar"]
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

func testAccDataInstallWithListArg(attr string, value []string) string {
	return fmt.Sprintf(`
		data "flux_install" "main" {
			target_path = "staging-cluster"
			%s = ["%s"]
		}
	`, attr, strings.Join(value, "\",\""))
}
