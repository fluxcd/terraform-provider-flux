package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccSync_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSyncBasicDataSource,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.flux_sync.main", "content"),
					resource.TestCheckResourceAttr("data.flux_sync.main", "path", "staging-cluster/flux-system/gotk-sync.yaml"),
				),
			},
		},
	})
}

const testAccSyncBasicDataSource = `
data "flux_sync" "main" {
  target_path = "staging-cluster"
  url = "ssh://git@example.com"
}
`
