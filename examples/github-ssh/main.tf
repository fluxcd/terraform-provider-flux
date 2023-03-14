terraform {
  required_version = ">=1.1.5"

  required_providers {
    flux = {
      source  = "fluxcd/flux"
      version = ">=0.24.2"
    }
    kind = {
      source  = "tehcyx/kind"
      version = ">=0.0.16"
    }
    github = {
      source  = "integrations/github"
      version = ">=5.18.0"
    }
  }
}

provider "github" {
  owner = var.github_org
  token = var.github_token
}

resource "tls_private_key" "this" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

data "github_repository" "this" {
  full_name = "${var.github_org}/${var.github_repository}"
}

resource "github_repository_deploy_key" "this" {
  title      = "Flux"
  repository = var.github_repository
  key        = tls_private_key.this.public_key_openssh
  read_only  = "false"
}

provider "kind" {}

resource "kind_cluster" "this" {
  name = "flux-e2e"
}

provider "flux" {
  kubernetes = {
    host                   = kind_cluster.this.endpoint
    client_certificate     = kind_cluster.this.client_certificate
    client_key             = kind_cluster.this.client_key
    cluster_ca_certificate = kind_cluster.this.cluster_ca_certificate
  }
  git = {
    url = "ssh://git@github.com/${data.github_repository.this.full_name}.git"
    ssh = {
      username    = "git"
      private_key = tls_private_key.this.private_key_pem
    }
  }
}

resource "flux_bootstrap_git" "this" {
  depends_on = [github_repository_deploy_key.this]

  path = "clusters/my-cluster"
}
