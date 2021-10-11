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
					resource.TestCheckResourceAttr(resourceName, "interval", "1"),
					resource.TestCheckResourceAttr(resourceName, "path", "staging-cluster/flux-system/gotk-sync.yaml"),
					resource.TestCheckResourceAttr(resourceName, "kustomize_path", "staging-cluster/flux-system/kustomization.yaml"),
					resource.TestCheckResourceAttr(resourceName, "content", testAccDataSyncBasicExpectedContent),
				),
			},
			// Ensure attribute value changes are propagated correctly into the state
			{
				Config: testAccDataSyncWithArg("namespace", "test-system"),
				Check:  resource.TestCheckResourceAttr(resourceName, "namespace", "test-system"),
			},
			{
				Config: testAccDataSyncWithArg("secret", "my-secret"),
				Check:  resource.TestCheckResourceAttr(resourceName, "secret", "my-secret"),
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
				Config: testAccDataSyncWithArg("tag", "1.0.0"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tag", "1.0.0"),
					resource.TestCheckResourceAttr(resourceName, "content", testAccDataSyncContentWithGitRefArg(`tag`, `1.0.0`)),
				),
			},
			{
				Config: testAccDataSyncWithArg("semver", ">1.0.0"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "semver", ">1.0.0"),
					resource.TestCheckResourceAttr(resourceName, "content", testAccDataSyncContentWithGitRefArg(`semver`, `'>1.0.0'`)),
				),
			},
			{
				Config: testAccDataSyncWithArg("commit", "ed8459bd047d0cfff48612c64cc994b1b5dfea23"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "commit", "ed8459bd047d0cfff48612c64cc994b1b5dfea23"),
					resource.TestCheckResourceAttr(resourceName, "content", testAccDataSyncContentWithGitRefArg(`commit`, `ed8459bd047d0cfff48612c64cc994b1b5dfea23`)),
				),
			},
			{
				Config: testAccDataSyncWithArg("interval", "5"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "interval", "5"),
					resource.TestCheckResourceAttr(resourceName, "content", testAccDataSyncIntervalExpectedContent),
				),
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
	testAccDataSyncBasicExpectedContent = `---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: flux-system
  namespace: flux-system
spec:
  interval: 1m0s
  ref:
    branch: main
  secretRef:
    name: flux-system
  url: ssh://git@example.com
---
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: flux-system
  namespace: flux-system
spec:
  interval: 10m0s
  path: ./staging-cluster
  prune: true
  sourceRef:
    kind: GitRepository
    name: flux-system
`
	testAccDataSyncIntervalExpectedContent = `---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: flux-system
  namespace: flux-system
spec:
  interval: 5m0s
  ref:
    branch: main
  secretRef:
    name: flux-system
  url: ssh://git@example.com
---
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: flux-system
  namespace: flux-system
spec:
  interval: 10m0s
  path: ./staging-cluster
  prune: true
  sourceRef:
    kind: GitRepository
    name: flux-system
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

func testAccDataSyncContentWithGitRefArg(attr, value string) string {
	return fmt.Sprintf(`---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: flux-system
  namespace: flux-system
spec:
  interval: 1m0s
  ref:
    branch: main
    %s: %s
  secretRef:
    name: flux-system
  url: ssh://git@example.com
---
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: flux-system
  namespace: flux-system
spec:
  interval: 10m0s
  path: ./staging-cluster
  prune: true
  sourceRef:
    kind: GitRepository
    name: flux-system
`, attr, value)
}
