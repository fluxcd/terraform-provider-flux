provider "flux" {
  kubernetes = {
    host                   = kind_cluster.this.endpoint
    client_certificate     = kind_cluster.this.client_certificate
    client_key             = kind_cluster.this.client_key
    cluster_ca_certificate = kind_cluster.this.cluster_ca_certificate
  }
  git = {
    url = "ssh://git@${var.forgejo_host}:${var.forgejo_port}/${var.forgejo_org}/${var.forgejo_repository}.git"
    ssh = {
      username    = "git"
      private_key = tls_private_key.ed25519.private_key_openssh
    }
  }
}

provider "forgejo" {
  host      = "https://${var.forgejo_host}:${var.forgejo_port}"
  api_token = var.forgejo_token
}

provider "kind" {}
