package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccInstall_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccInstallBasicDataSource,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.flux_install.main", "content"),
					resource.TestCheckResourceAttr("data.flux_install.main", "path", "staging-cluster/flux-system/gotk-components.yaml"),
				),
			},
		},
	})
}

const testAccInstallBasicDataSource = `
data "flux_install" "main" {
  target_path = "staging-cluster"
}
`
