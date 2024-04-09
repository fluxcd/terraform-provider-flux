terraform {
  required_version = ">= 1.7.0"

  required_providers {
    flux = {
      source  = "fluxcd/flux"
      version = ">= 1.2"
    }
    github = {
      source  = "integrations/github"
      version = ">= 6.1"
    }
    kind = {
      source  = "tehcyx/kind"
      version = ">= 0.4"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.27"
    }
    tls = {
      source  = "hashicorp/tls"
      version = ">= 4.0"
    }
  }
}

# ========================================================================
# Construct KinD cluster
# ========================================================================

resource "kind_cluster" "this" {
  name = "flux-e2e"
}

# ========================================================================
# Initialise a Github project
# ========================================================================

resource "github_repository" "this" {
  name        = var.github_repository
  description = var.github_repository
  visibility  = "public"
  auto_init   = true # This is extremely important as flux_bootstrap_git will not work without a repository that has been initialised
}

# ========================================================================
# Add deploy key to GitHub repository
# ========================================================================

resource "tls_private_key" "flux" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

resource "github_repository_deploy_key" "this" {
  title      = "Flux"
  repository = github_repository.this.name
  key        = tls_private_key.flux.public_key_openssh
  read_only  = "false"
}

# ========================================================================
# Manage the SSH keypair flux uses to authenticate with GitHub
# ========================================================================

resource "kubernetes_namespace" "flux_system" {
  metadata {
    name = "flux-system"
  }

  lifecycle {
    ignore_changes = [metadata]
  }
}

resource "kubernetes_secret" "ssh_keypair" {
  metadata {
    name      = "flux-system"
    namespace = "flux-system"
  }

  type = "Opaque"

  data = {
    "identity.pub" = tls_private_key.flux.public_key_openssh
    "identity"     = tls_private_key.flux.private_key_pem
    "known_hosts"  = "github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg="
  }

  depends_on = [kubernetes_namespace.flux_system]
}

# ========================================================================
# Bootstrap KinD cluster
# ========================================================================

resource "flux_bootstrap_git" "this" {
  depends_on = [github_repository_deploy_key.this, kubernetes_secret.ssh_keypair]

  path                    = "clusters/my-cluster"
  disable_secret_creation = true
}
