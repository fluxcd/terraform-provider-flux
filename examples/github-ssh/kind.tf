provider "kind" {}

resource "kind_cluster" "this" {
  name = "flux-e2e"
}
