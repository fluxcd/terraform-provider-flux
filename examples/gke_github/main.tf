terraform {
  required_version = ">= 0.12.10"

  required_providers {
    github = {
      source  = "hashicorp/github"
      version = ">= 4.5.1"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.0.2"
    }
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = ">= 1.10.0"
    }
    flux = {
      source  = "fluxcd/flux"
      version = ">= 0.0.13"
    }
  }
}

# Flux
provider "flux" {}

data "flux_install" "main" {
  target_path = var.target_path
}

data "flux_sync" "main" {
  target_path = var.target_path
  url         = "ssh://git@github.com/${var.organization}/${var.repository_name}.git"
  branch      = var.branch
}

# https://registry.terraform.io/modules/terraform-google-modules/kubernetes-engine/google/latest/submodules/auth
module "gke_auth" {
  source               = "terraform-google-modules/kubernetes-engine/google//modules/auth"
  project_id           = var.project_id
  cluster_name         = var.cluster_name
  location             = var.cluster_region
  use_private_endpoint = var.use_private_endpoint
}

provider "kubernetes" {
  cluster_ca_certificate = module.gke_auth.cluster_ca_certificate
  host                   = module.gke_auth.host
  token                  = module.gke_auth.token
}

provider "kubectl" {
  cluster_ca_certificate = module.gke_auth.cluster_ca_certificate
  host                   = module.gke_auth.host
  token                  = module.gke_auth.token
  load_config_file       = false
}

# Kubernetes
resource "kubernetes_namespace" "flux_system" {
  metadata {
    name = var.flux_namespace
  }

  lifecycle {
    ignore_changes = [
      metadata[0].labels,
    ]
  }
}

data "kubectl_file_documents" "install" {
  content = data.flux_install.main.content
}

resource "kubectl_manifest" "install" {
  depends_on = [kubernetes_namespace.flux_system]
  for_each   = { for v in data.kubectl_file_documents.install.documents : sha1(v) => v }
  yaml_body  = each.value
}

data "kubectl_file_documents" "sync" {
  content = data.flux_sync.main.content
}

resource "kubectl_manifest" "sync" {
  depends_on = [kubectl_manifest.install, kubernetes_namespace.flux_system]
  for_each   = { for v in data.kubectl_file_documents.sync.documents : sha1(v) => v }
  yaml_body  = each.value
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
    known_hosts = local.known_hosts
    password    = var.github_token
  }
}

# Github
provider "github" {
  token        = var.github_token
  organization = var.organization
}

resource "github_repository_file" "install" {
  repository          = var.repository_name
  file                = data.flux_install.main.path
  content             = data.flux_install.main.content
  branch              = var.branch
  overwrite_on_create = true
}

resource "github_repository_file" "sync" {
  repository          = var.repository_name
  file                = data.flux_sync.main.path
  content             = data.flux_sync.main.content
  branch              = var.branch
  overwrite_on_create = true
}

resource "github_repository_file" "kustomize" {
  repository          = var.repository_name
  file                = data.flux_sync.main.kustomize_path
  content             = data.flux_sync.main.kustomize_content
  branch              = var.branch
  overwrite_on_create = true
}
