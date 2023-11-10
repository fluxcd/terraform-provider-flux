provider "flux" {
  kubernetes = {
    host                   = kind_cluster.this.endpoint
    client_certificate     = kind_cluster.this.client_certificate
    client_key             = kind_cluster.this.client_key
    cluster_ca_certificate = kind_cluster.this.cluster_ca_certificate
  }
  git = {
    url = "ssh://git@github.com/${var.github_org}/${var.github_repository}.git"
    ssh = {
      username       = "git"
      private_key    = tls_private_key.flux.private_key_pem
    }
    gpg_key_ring   = var.flux_gpg_key_ring
    gpg_key_id     = var.flux_gpg_key_id
    gpg_passphrase = var.flux_gpg_passphrase
  }
}

resource "flux_bootstrap_git" "this" {
  depends_on = [github_repository_deploy_key.this]

  path = "clusters/my-cluster"
}
