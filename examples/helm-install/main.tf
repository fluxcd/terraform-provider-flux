# Copyright (c) The Flux authors
# SPDX-License-Identifier: Apache-2.0

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
    helm = {
      source  = "hashicorp/helm"
      version = ">= 2.12"
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

# ===============================================
# Bootstrap KinD cluster using flux helm chart
# ===============================================

# Ref: https://github.com/fluxcd-community/helm-charts/tree/main/charts/flux2
resource "helm_release" "this" {
  repository = "https://fluxcd-community.github.io/helm-charts"
  chart      = "flux2"
  version    = "2.12.4"

  name             = "flux2"
  namespace        = "flux-system"
  create_namespace = true

  # Allows us to upgrade the version of flux from another repo and terraform not roll it back.
  lifecycle {
    ignore_changes = [
      version
    ]
  }
}
