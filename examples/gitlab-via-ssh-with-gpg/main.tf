terraform {
  required_version = ">= 1.7.0"

  required_providers {
    flux = {
      source  = "fluxcd/flux"
      version = ">= 1.2"
    }
    gitlab = {
      source  = "gitlabhq/gitlab"
      version = ">= 16.10"
    }
    kind = {
      source  = "tehcyx/kind"
      version = ">= 0.4"
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

resource "kind_cluster" "this" {
  name = "flux-e2e"
}

# ==========================================
# Initialise a Gitlab project
# ==========================================

resource "gitlab_project" "this" {
  name                   = var.gitlab_project
  description            = "flux-bootstrap"
  visibility_level       = "public"
  initialize_with_readme = true # This is extremely important as Flux expects an initialised repository
}

# ==========================================
# Add deploy key to Gitlab repository
# ==========================================

resource "tls_private_key" "flux" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

resource "gitlab_deploy_key" "this" {
  project  = gitlab_project.this.path_with_namespace
  title    = "Flux"
  key      = tls_private_key.flux.public_key_openssh
  can_push = true
}

# ==========================================
# Bootstrap KinD cluster
# ==========================================

resource "flux_bootstrap_git" "this" {
  depends_on = [gitlab_deploy_key.this]

  path = "clusters/my-cluster"
}
