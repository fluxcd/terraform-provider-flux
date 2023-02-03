resource "flux_bootstrap_git" "this" {
  url  = var.repository_ssh_url
  path = "clusters/my-cluster"
  ssh = {
    username    = "git"
    private_key = var.private_key_pem
  }
  kustomization_override = file("${path.module}/kustomization.yaml")
}
