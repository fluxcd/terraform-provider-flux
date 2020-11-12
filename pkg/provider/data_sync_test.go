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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSync_basic(t *testing.T) {
	resourceName := "data.flux_sync.main"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				// Without required target_path set
				Config:      testAccDataSyncMissingTargetPath,
				ExpectError: regexp.MustCompile(`The argument "target_path" is required, but no definition was found\.`),
			},
			{
				// Without required url set
				Config:      testAccDataSyncMissingURL,
				ExpectError: regexp.MustCompile(`The argument "url" is required, but no definition was found\.`),
			},
			{
				// Incorrect url syntax
				Config:      testAccDataSyncInCorrectURL,
				ExpectError: regexp.MustCompile(`Error: expected "url" to have a url with schema of: "http,https,ssh", got ftp://git@example.com`),
			},
			{
				// Check default values
				Config: testAccDataSyncBasic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "content"),
					resource.TestCheckResourceAttrSet(resourceName, "kustomize_content"),
					resource.TestCheckResourceAttr(resourceName, "namespace", "flux-system"),
					resource.TestCheckResourceAttr(resourceName, "url", "ssh://git@example.com"),
					resource.TestCheckResourceAttr(resourceName, "branch", "main"),
					resource.TestCheckResourceAttr(resourceName, "target_path", "staging-cluster"),
					resource.TestCheckResourceAttr(resourceName, "interval", fmt.Sprintf("%d", time.Minute)),
					resource.TestCheckResourceAttr(resourceName, "path", "staging-cluster/flux-system/gotk-sync.yaml"),
					resource.TestCheckResourceAttr(resourceName, "kustomize_path", "staging-cluster/flux-system/kustomization.yaml"),
				),
			},
			// Ensure attribute value changes are propagated correctly into the state
			{
				Config: testAccDataSyncWithArg("namespace", "test-system"),
				Check:  resource.TestCheckResourceAttr(resourceName, "namespace", "test-system"),
			},
			{
				Config: testAccDataSyncWithArg("name", "my-flux-install"),
				Check:  resource.TestCheckResourceAttr(resourceName, "name", "my-flux-install"),
			},
			{
				Config: testAccDataSyncWithArg("branch", "develop"),
				Check:  resource.TestCheckResourceAttr(resourceName, "branch", "develop"),
			},
			{
				Config: testAccDataSyncWithArg("interval", "90000000000"),
				Check:  resource.TestCheckResourceAttr(resourceName, "interval", "90000000000"),
			},
		},
	})
}

const (
	testAccDataSyncMissingTargetPath = `data "flux_sync" "main" {  url = "ssh://git@example.com" }`
	testAccDataSyncMissingURL        = `data "flux_sync" "main" {  target_path = "test" }`
	testAccDataSyncBasic             = `
		data "flux_sync" "main" {
			target_path = "staging-cluster"
			url = "ssh://git@example.com"
		}
	`
	testAccDataSyncInCorrectURL = `
		data "flux_sync" "main" {
			target_path = "staging-cluster"
			url = "ftp://git@example.com"
		}
	`
)

func testAccDataSyncWithArg(attr string, value string) string {
	return fmt.Sprintf(`
		data "flux_sync" "main" {
			target_path = "staging-cluster"
			url = "ssh://git@example.com"
			%s = %q
		}
	`, attr, value)
}
