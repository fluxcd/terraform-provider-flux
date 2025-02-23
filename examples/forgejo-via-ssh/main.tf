terraform {
  required_version = ">= 1.7.0"

  required_providers {
    flux = {
      source  = "fluxcd/flux"
      version = ">= 1.5"
    }
    forgejo = {
      source  = "svalabs/forgejo"
      version = ">= 0.2"
    }
    kind = {
      source  = "tehcyx/kind"
      version = ">= 0.8"
    }
    tls = {
      source  = "hashicorp/tls"
      version = ">= 4.0"
    }
  }
}

# ==========================================
# Construct KinD cluster
# ==========================================

# https://registry.terraform.io/providers/tehcyx/kind/latest/docs/resources/cluster
resource "kind_cluster" "this" {
  name = "flux"
}

# ==========================================
# Initialise a Forgejo repository
# ==========================================

# https://registry.terraform.io/providers/svalabs/forgejo/latest/docs/resources/repository
resource "forgejo_repository" "this" {
  owner       = var.forgejo_org
  name        = var.forgejo_repository
  description = "Flux repository"
  private     = true
  auto_init   = true # This is extremely important as flux_bootstrap_git will not work without a repository that has been initialised
}

# ==========================================
# Add deploy key to GitHub repository
# ==========================================

# https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key
resource "tls_private_key" "ed25519" {
  algorithm = "ED25519"
}

# https://registry.terraform.io/providers/svalabs/forgejo/latest/docs/resources/deploy_key
resource "forgejo_deploy_key" "this" {
  title         = "flux"
  repository_id = forgejo_repository.this.id
  key           = trimspace(tls_private_key.ed25519.public_key_openssh)
  read_only     = false
}

# ==========================================
# Bootstrap KinD cluster
# ==========================================

# https://registry.terraform.io/providers/fluxcd/flux/latest/docs/resources/bootstrap_git
resource "flux_bootstrap_git" "this" {
  depends_on = [forgejo_deploy_key.this]

  embedded_manifests = true
  path               = "clusters/my-cluster"
}
