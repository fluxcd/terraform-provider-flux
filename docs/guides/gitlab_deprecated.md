---
page_title: "Bootstrap a cluster with GitLab and flux_install"
subcategory: "Deprecated"
description: |-
    An example of how to bootstrap a Kubernetes cluster and sync it with a GitLab repository.
---

# Bootstrap a cluster with GitLab

In order to follow the guide you'll need a GitLab account and a [personal access token](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)
that can create repositories.

Create the staging cluster using Kubernetes kind or set the kubectl context to an existing cluster:
```
kind create cluster --name staging
kubectl cluster-info --context kind-staging
```

With the following variables default values are set for all but `gitlab_owner` and `gitlab_token`.
```terraform
variable "gitlab_owner" {
  description = "gitlab owner"
  type        = string
}

variable "gitlab_token" {
  description = "gitlab token"
  type        = string
  sensitive   = true
}

variable "repository_name" {
  description = "gitlab repository name"
  type        = string
  default     = "test-provider"
}

variable "repository_visibility" {
  description = "how visible is the gitlab repo"
  type        = string
  default     = "private"
}

variable "branch" {
  description = "branch name"
  type        = string
  default     = "main"
}

variable "target_path" {
  description = "flux sync target path"
  type        = string
  default     = "staging-cluster"
}
```

You can set these in the terminal that you are running your terraform command by exporting variables.
```shell
export TF_VAR_gitlab_owner=<owner>
export TF_VAR_gitlab_token=<token>
```

By using the GitLab provider to create a repository you can commit the manifests given by the
data sources `flux_install` and `flux_sync`. The cluster has been successfully provisioned
after the same manifests are applied to it.
```terraform
terraform {
  required_version = ">= 1.1.5"

  required_providers {
    gitlab = {
      source  = "gitlabhq/gitlab"
      version = ">= 3.11.1"
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
      version = ">= 1.0.0-rc.1"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "3.1.0"
    }
  }
}

provider "flux" {}

provider "kubectl" {}

provider "kubernetes" {
  config_path = "~/.kube/config"
}

provider "gitlab" {
  token = var.gitlab_token
}

# SSH
locals {
  known_hosts = "gitlab.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBFSMqzJeV9rUzU4kWitGjeR4PWSa29SPqJ1fVkhtj3Hw9xjLVXVYrU9QlYWrOLXBpQ6KWjbjTDTdDkoohFzgbEY="
}

resource "tls_private_key" "main" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

# Flux
data "flux_install" "main" {
  target_path = var.target_path
}

data "flux_sync" "main" {
  target_path = var.target_path
  url         = "ssh://git@gitlab.com/${var.gitlab_owner}/${var.repository_name}.git"
  branch      = var.branch
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

data "kubectl_file_documents" "sync" {
  content = data.flux_sync.main.content
}

locals {
  install = [for v in data.kubectl_file_documents.install.documents : {
    data : yamldecode(v)
    content : v
    }
  ]
  sync = [for v in data.kubectl_file_documents.sync.documents : {
    data : yamldecode(v)
    content : v
    }
  ]
}

resource "kubectl_manifest" "install" {
  for_each   = { for v in local.install : lower(join("/", compact([v.data.apiVersion, v.data.kind, lookup(v.data.metadata, "namespace", ""), v.data.metadata.name]))) => v.content }
  depends_on = [kubernetes_namespace.flux_system]
  yaml_body  = each.value
}

resource "kubectl_manifest" "sync" {
  for_each   = { for v in local.sync : lower(join("/", compact([v.data.apiVersion, v.data.kind, lookup(v.data.metadata, "namespace", ""), v.data.metadata.name]))) => v.content }
  depends_on = [kubernetes_namespace.flux_system]
  yaml_body  = each.value
}

resource "kubernetes_secret" "main" {
  depends_on = [kubectl_manifest.install]

  metadata {
    name      = data.flux_sync.main.secret
    namespace = data.flux_sync.main.namespace
  }

  data = {
    identity       = tls_private_key.main.private_key_pem
    "identity.pub" = tls_private_key.main.public_key_pem
    known_hosts    = local.known_hosts
  }
}

# Gitlab
resource "gitlab_project" "main" {
  name                   = var.repository_name
  visibility_level       = var.repository_visibility
  initialize_with_readme = true
  default_branch         = var.branch
}

resource "gitlab_deploy_key" "main" {
  title   = "staging-cluster"
  project = gitlab_project.main.id
  key     = tls_private_key.main.public_key_openssh

  depends_on = [gitlab_project.main]
}

resource "gitlab_repository_file" "install" {
  project        = gitlab_project.main.id
  branch         = gitlab_project.main.default_branch
  file_path      = data.flux_install.main.path
  content        = base64encode(data.flux_install.main.content)
  commit_message = "Add ${data.flux_install.main.path}"

  depends_on = [gitlab_project.main]
}

resource "gitlab_repository_file" "sync" {
  project        = gitlab_project.main.id
  branch         = gitlab_project.main.default_branch
  file_path      = data.flux_sync.main.path
  content        = base64encode(data.flux_sync.main.content)
  commit_message = "Add ${data.flux_sync.main.path}"

  depends_on = [gitlab_repository_file.install]
}

resource "gitlab_repository_file" "kustomize" {
  project        = gitlab_project.main.id
  branch         = gitlab_project.main.default_branch
  file_path      = data.flux_sync.main.kustomize_path
  content        = base64encode(data.flux_sync.main.kustomize_content)
  commit_message = "Add ${data.flux_sync.main.kustomize_path}"

  depends_on = [gitlab_repository_file.sync]
}
```
