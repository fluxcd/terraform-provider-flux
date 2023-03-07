resource "flux_bootstrap_git" "this" {
  path                   = "clusters/my-cluster"
  kustomization_override = file("${path.module}/kustomization.yaml")
}
