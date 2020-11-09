terraform {
  required_version = ">= 0.13"

  required_providers {
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

provider "kubectl" {
  apply_retry_count = 15
}

# Flux
data "flux_install" "main" {
  target_path = var.target_path
}

# Kubernetes
data "kubectl_file_documents" "install" {
  content = data.flux_install.main.content
}

resource "kubectl_manifest" "install" {
  for_each  = { for v in data.kubectl_file_documents.install.documents : sha1(v) => v }
  yaml_body = each.value
}
