---
subcategory: ""
page_title: "Migrating to bootstrap resource"
description: |-
    Guide to migrate from boostraping Flux with datasources to using the bootstrap resource. 
---

# Migrating to bootstrap resource

The original method to bootstrap Flux with Terraform was designed around using other Terraform providers existing functionality instead of implementing the same functionality in the Flux provider. This seemed initially like a good idea, with simple Flux configurations, but turned into a very complex solution as time progressed. There were a couple of flaws with the original solution, on top of being difficult for most to understand. Dependency complexity between resources was high which made cyclical dependency common. The method at which API Versions were handled when applying resources made it so that there was at times a risk that old custom resource versions were removed before the new one could be applied. Lastly destroying the Terraform resources were not guaranteed to work as Terraform had no idea which order resources should be removed to allow Flux to clean up finalizers, causing resources to get stuck in a limbo state where they cannot be removed. It was for these reasons decided to refactor the provider and switch to using a single Terraform resource which would manage the full bootstrapping process for Flux, in a similar manner to how the Flux CLI does.

> Make sure that you have a backup of your state before modifying the state. If you are using versioned remote state you will be fine as you can always rollback. If not please run `terraform state pull > terraform.state` to save a local copy of your Terraform state. That way if something goes wrong along the way you can start from the beginning by pushing the local state with `terraform state push terraform.state`.

An existing Flux installation can be migrated from the old datasource method to the new resource method with a couple of state modifications. The old resources have to be removed while the existing installation has to be imported into the new resource. To simplify the process it is suggested that you do not upgrade the Flux version during the migration. Flux installations may look a bit different from each other depending on configuration and git provider used. This guide will use a basic GitHub bootstrap configuration as an example, but the import process should be similar for other git providers. The resources that have to be removed from the state may however differ between different git providers.

```hcl
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

terraform {
  required_version = ">= 1.1.5"
  required_providers {
    github = {
      source  = "integrations/github"
      version = ">= 5.9.1"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.16.0"
    }
    kubectl = {
      source  = "gavinbunney/kubectl"
      version = ">= 1.14.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = ">= 4.0.4"
    }
    flux = {
      source = "fluxcd/flux"
      version = ">= 0.22.0"
    }
  }
}

provider "flux" {}

provider "kubectl" {}

provider "kubernetes" {
  config_path = "~/.kube/config"
}

provider "github" {
  owner = var.github_owner
  token = var.github_token
}

locals {
  known_hosts = "github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg="
}

resource "tls_private_key" "this" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

data "flux_install" "this" {
  target_path = var.target_path
}

data "flux_sync" "this" {
  target_path = var.target_path
  url         = "ssh://git@github.com/${var.github_owner}/${var.repository_name}.git"
  branch      = var.branch
}

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
  content = data.flux_install.this.content
}

data "kubectl_file_documents" "sync" {
  content = data.flux_sync.this.content
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

resource "kubernetes_secret" "this" {
  depends_on = [kubectl_manifest.install]

  metadata {
    name      = data.flux_sync.this.secret
    namespace = data.flux_sync.this.namespace
  }

  data = {
    identity       = tls_private_key.this.private_key_pem
    "identity.pub" = tls_private_key.this.public_key_pem
    known_hosts    = local.known_hosts
  }
}

resource "github_repository" "this" {
  name       = var.repository_name
  visibility = var.repository_visibility
  auto_init  = true
}

resource "github_branch_default" "this" {
  repository = github_repository.this.name
  branch     = var.branch
}

resource "github_repository_deploy_key" "this" {
  title      = "staging-cluster"
  repository = github_repository.this.name
  key        = tls_private_key.this.public_key_openssh
  read_only  = false
}

resource "github_repository_file" "install" {
  repository = github_repository.this.name
  file       = data.flux_install.this.path
  content    = data.flux_install.this.content
  branch     = var.branch
}

resource "github_repository_file" "sync" {
  repository = github_repository.this.name
  file       = data.flux_sync.this.path
  content    = data.flux_sync.this.content
  branch     = var.branch
}

resource "github_repository_file" "kustomize" {
  repository = github_repository.this.name
  file       = data.flux_sync.this.kustomize_path
  content    = data.flux_sync.this.kustomize_content
  branch     = var.branch
}
```

The old method to bootstrap Flux is extensive to say the least. The same configuration can be defined in a single resource with the new bootstrap method. Replace the old Terraform configuration with the following.

```hcl
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

terraform {
  required_version = ">= 1.1.5"
  required_providers {
    github = {
      source  = "integrations/github"
      version = ">= 5.9.1"
    }
    tls = {
      source  = "hashicorp/tls"
      version = ">= 4.0.4"
    }
    flux = {
      source = "fluxcd/flux"
      version = ">= 0.22.0"
    }
  }
}

provider "github" {
  owner = var.github_owner
  token = var.github_token
}

provider "flux" {
  kubernetes = {
    config_path = "~/.kube/config"
  }
  git = {
    url  = "ssh://git@github.com/${github_repository.this.full_name}.git"
    ssh = {
      username    = "git"
      private_key = tls_private_key.this.private_key_pem
    }
  }
}

resource "tls_private_key" "this" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

resource "github_repository" "this" {
  name       = var.repository_name
  visibility = var.repository_visibility
  auto_init  = true
}

resource "github_branch_default" "this" {
  repository = github_repository.this.name
  branch     = var.branch
}

resource "github_repository_deploy_key" "this" {
  title      = "staging-cluster"
  repository = github_repository.this.name
  key        = tls_private_key.this.public_key_openssh
  read_only  = false
}

resource "flux_bootstrap_git" "this" {
  depends_on = [github_repository_deploy_key.this]

  path = "staging-cluster"
}
```

If you were to run apply now, Terraform would attempt to create a new Flux bootstrap resource. We don't want this as Flux has already been bootstrapped and is running inside of the cluster. Instead we want to import the existing Flux deployment into the new resource. The provider will need access to the Kubernetes cluster when import is run. The only parameter that needs to be passed is the namespace in which Flux is installed in. The provider will with this information retrieve the required configuration like repository URL and credentials from Kubernetes. It will then check the files in git to make sure that they match what is expected.

```shell
tf import flux_bootstrap_git.this flux-system
```

The old Terraform resources have to be removed from the Terraform state before apply can be run to complete the migration, as there are now two separate Terraform resources which are managing the same Kubernetes resources. Removing them from the state will make Terraform forget about them without destroying them.

```shell
terraform state rm kubernetes_namespace.flux_system
terraform state rm kubectl_manifest.install
terraform state rm kubectl_manifest.sync
terraform state rm kubernetes_secret.this
terraform state rm data.flux_install.this 
terraform state rm data.flux_sync.this
terraform state rm data.kubectl_file_documents.install
terraform state rm data.kubectl_file_documents.sync
terraform state rm github_repository_file.install
terraform state rm github_repository_file.kustomize
terraform state rm github_repository_file.sync
```

The last step in the migration is to run Terraform apply. The first time you run this after that import may result in some additional configuration parameters being set. These are most likely default values which is normal and expected. Proceed with the apply unless Terraform wants to replace the resource all together or is going to make substantial incorrect changes to the `repository_files` parameter.

```shell
terraform apply
```

The migration is now complete and you can use the new Terraform resource to manage Flux.
