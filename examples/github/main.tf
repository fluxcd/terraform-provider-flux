terraform {
  required_version = ">= 0.13"

  required_providers {
    github = {
      source  = "hashicorp/github"
      version = ">= 3.1.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 1.13.3"
    }
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = ">= 1.9.1"
    }
    flux = {
      source  = "fluxcd/flux"
      version = ">= 0.0.1"
    }
  }
}

provider "flux" {}

provider "kubectl" {}

provider "kubernetes" {}

provider "github" {
  owner = var.github_owner
  token = var.github_token
}

# SSH
resource "tls_private_key" "main" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

# Flux
data "flux_install" "main" {
  target_path = var.target_path
}

data "flux_sync" "main" {
  target_path = var.target_path
  url         = "ssh://git@github.com/${var.github_owner}/${var.repository_name}.git"
}

# Kubernetes
data "kubectl_file_documents" "install" {
  content = data.flux_install.main.content
}

resource "kubectl_manifest" "install" {
  for_each  = { for v in data.kubectl_file_documents.install.documents : sha1(v) => v }
  yaml_body = each.value
}

data "kubectl_file_documents" "sync" {
  content = data.flux_sync.main.content
}

resource "kubectl_manifest" "sync" {
  depends_on = [kubectl_manifest.install]

  for_each  = { for v in data.kubectl_file_documents.sync.documents : sha1(v) => v }
  yaml_body = each.value
}

locals {
  known_hosts = "github.com ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+PXYPCPy6rbTrTtw7PHkccKrpp0yVhp5HdEIcKr6pLlVDBfOLX9QUsyCOV0wzfjIJNlGEYsdlLJizHhbn2mUjvSAHQqZETYP81eFzLQNnPHt4EVVUh7VfDESU84KezmD5QlWpXLmvU31/yMf+Se8xhHTvKSCZIFImWwoG6mbUoWf9nzpIoaSjB+weqqUUmpaaasXVal72J+UX2B+2RPW3RcT0eOzQgqlJL3RKrTJvdsjE3JEAvGq3lGHSZXy28G3skua2SmVi/w4yCE6gbODqnTWlg7+wC604ydGXA8VJiS5ap43JXiUFFAaQ=="
}

resource "kubernetes_secret" "main" {
  depends_on = [kubectl_manifest.install]

  metadata {
    name      = data.flux_sync.main.name
    namespace = data.flux_sync.main.namespace
  }

  data = {
    identity       = tls_private_key.main.private_key_pem
    "identity.pub" = tls_private_key.main.public_key_pem
    known_hosts    = local.known_hosts
  }
}

# GitHub
resource "github_repository" "main" {
  name           = var.repository_name
  visibility     = var.repository_visibility
  default_branch = var.branch
  auto_init      = true
}

resource "github_repository_deploy_key" "main" {
  title      = "flux2"
  repository = github_repository.main.name
  key        = tls_private_key.main.public_key_openssh
  read_only  = true
}

resource "github_repository_file" "install" {
  repository = github_repository.main.name
  file       = data.flux_install.main.path
  content    = data.flux_install.main.content
  branch     = var.branch
}

resource "github_repository_file" "sync" {
  repository = github_repository.main.name
  file       = data.flux_sync.main.path
  content    = data.flux_sync.main.content
  branch     = var.branch
}
