terraform {
  required_version = "= 1.7.0"

  required_providers {
    flux = {
      source  = "fluxcd/flux"
      version = "1.2.3"
    }
    github = {
      source  = "integrations/github"
      version = "6.1.0"
    }
    kind = {
      source  = "tehcyx/kind"
      version = "0.4.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "4.0.5"
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
# Add deploy key to GitHub repository
# ==========================================

resource "tls_private_key" "flux" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

resource "github_repository_deploy_key" "this" {
  title      = "Flux"
  repository = var.github_repository
  key        = tls_private_key.flux.public_key_openssh
  read_only  = "false"
}

# ==========================================
# Bootstrap KinD cluster
# ==========================================

resource "flux_bootstrap_git" "this" {
  depends_on = [github_repository_deploy_key.this]

  version = var.flux_version
  path    = "clusters/my-cluster"
  components_extra = [
    "image-reflector-controller",
    "image-automation-controller"
  ]
  kustomization_override = file("${path.root}/resources/flux-kustomization-patch.yaml")
}
