# Copyright (c) The Flux authors
# SPDX-License-Identifier: Apache-2.0

provider "flux" {
  kubernetes = {
    host                   = kind_cluster.this.endpoint
    client_certificate     = kind_cluster.this.client_certificate
    client_key             = kind_cluster.this.client_key
    cluster_ca_certificate = kind_cluster.this.cluster_ca_certificate
  }
  git = {
    url = "ssh://git@gitlab.com/${data.gitlab_project.this.path_with_namespace}.git"
    ssh = {
      username    = "git"
      private_key = tls_private_key.flux.private_key_pem
    }
  }
}

provider "gitlab" {
  token = var.gitlab_token
}

provider "kind" {}
