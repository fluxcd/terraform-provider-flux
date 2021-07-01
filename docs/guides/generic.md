---
subcategory: ""
page_title: "Bootstrap a cluster with generic git repo"
description: |-
    An example of how to bootstrap a Kubernetes cluster and sync it with a generic git repository.
---

# Bootstrap a cluster with generic git repo

__Note:__ This is only a partial example focusing on pushing the flux manifests to a generic repo installing the manifests in the cluster is covered by the Github example.

In order to follow the guide you'll need a `generic` git account and a `deployment key` or `ssh key` configured
that can create write to the repository.

```terraform
variable "repository_url" {
  type        = string
  description = "The url for the git repo. Example: ssh://git@codeberg.org/<user>/fluxv2.git "
}

variable "branch" {
  type    = string
  default = "main"
}
```

```terraform
terraform {
  required_version = ">= 0.13"

  required_providers {
    flux = {
      source = "fluxcd/flux"
    }
    gitfile = {
      source  = "igal-s/gitfile"
      version = "1.0.0"
    }
  }
}

provider "flux" {}
provider "gitfile" {}

# Generate the flux manifest
data "flux_install" "main" {
  target_path = "cluster-init"
}

data "flux_sync" "main" {
  target_path = "cluster-init"
  url         = var.repository_url
}

# Git: Create the files and commit them to the repository
resource "gitfile_checkout" "main" {
  repo   = var.repository_url
  branch = var.branch
  path   = "${path.module}/git_checkouts/cluster-init"
}

resource "gitfile_file" "install" {
  checkout_dir = gitfile_checkout.example.path
  path         = data.flux_install.main.path
  contents     = data.flux_install.main.content
}

resource "gitfile_file" "sync" {
  checkout_dir = gitfile_checkout.example.path
  path         = data.flux_sync.main.path
  contents     = data.flux_sync.main.content
}

resource "gitfile_commit" "flux_install" {
  checkout_dir   = gitfile_checkout.example.path
  commit_message = "Added install from flux provider"
  handle         = gitfile_file.install.id
}

resource "gitfile_commit" "flux_sync" {
  checkout_dir   = gitfile_checkout.example.path
  commit_message = "Added sync from flux provider"
  handle         = gitfile_file.sync.id
}
```
