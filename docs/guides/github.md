---
subcategory: ""
page_title: "Bootstrap a cluster with GitHub"
description: |-
    An example of how to bootstrap a Kubernetes cluster and sync it with a GitHub repository.
---

# Bootstrap a cluster with GitHub

In order to follow the guide you'll need a GitHub account and a [personal access token](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token)
that can create repositories (check all permissions under repo).

Create the staging cluster using Kubernetes kind or set the kubectl context to an existing cluster:
```
kind create cluster --name staging
kubectl cluster-info --context kind-staging
```

With the following variables default values are set for all but `github_owner` and `github_token`.
```terraform
variable "github_owner" {
  type = string
}

variable "github_token" {
  type = string
}

variable "repository_name" {
  type    = string
  default = "test-provider"
}

variable "repository_visibility" {
  type    = string
  default = "private"
}

variable "branch" {
  type    = string
  default = "main"
}

variable "target_path" {
  type    = string
  default = "staging-cluster"
}
```

You can set these in the terminal that you are running your terraform command by exporting variables.
```shell
export TF_VAR_github_owner=<owner>
export TF_VAR_github_token=<token>
```

By using the GitHub provider to create a repository you can commit the manifests given by the
data sources `flux_install` and `flux_sync`. The cluster has been successfully after the same
manifests are applied to it.
```terraform
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
resource "kubernetes_namespace" "flux_system" {
  metadata {
    name = "flux-system"
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
  for_each  = { for v in data.kubectl_file_documents.install.documents : sha1(v) => v }
  depends_on = [kubernetes_namespace.flux_system]

  yaml_body = each.value
}

data "kubectl_file_documents" "sync" {
  content = data.flux_sync.main.content
}

resource "kubectl_manifest" "sync" {
  for_each  = { for v in data.kubectl_file_documents.sync.documents : sha1(v) => v }
  depends_on = [kubectl_manifest.install, kubernetes_namespace.flux_system]

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
  title      = "staging-cluster"
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

resource "github_repository_file" "kustomize" {
  repository = github_repository.main.name
  file       = data.flux_sync.main.kustomize_path
  content    = data.flux_sync.main.kustomize_content
  branch     = var.branch
}
```
