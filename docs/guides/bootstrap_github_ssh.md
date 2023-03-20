---
subcategory: ""
page_title: "Bootstrap with GitHub"
description: |-
	Install Flux and synchronize with GitHub.
---

# Bootstrap with GitHub

This guide will walk through how to install Flux into a Kubernetes cluster and configure it to synchronize from a GitHub repository. Begin with creating a GitHub repository, in this guide it will be named `fleet-infra`, which will be used by Flux.
Generate a [personal access token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) (PAT) with `repo` permissions, make sure to copy the generated token.

It is a good idea to use variables so that you do not accidentally store GitHub credentials in your Terraform configuration.

```terraform
variable "github_token" {
  sensitive = true
  type      = string
}

variable "github_org" {
  type = string
}

variable "github_repository" {
  type = string
}
```

Configure the required providers and their versions.

```terraform
terraform {
  required_version = ">=1.1.5"

  required_providers {
    flux = {
      source  = "fluxcd/flux"
      version = ">=0.24.2"
    }
    kind = {
      source  = "tehcyx/kind"
      version = ">=0.0.16"
    }
    github = {
      source  = "integrations/github"
      version = ">=5.18.0"
    }
  }
}
```

A Kind cluster is used as the target Kubernetes cluster where Flux is installed.

```terraform
provider "kind" {}

resource "kind_cluster" "this" {
  name = "flux-e2e"
}
```

The GitHub repository is created separatly so a datasource is used to get a reference to the repository. Creating GitHub repositories with Terraform is generally not a good idea as they could easily be removed. Additionally it is not possible to use the same repository for multiple environments if the repository is created with Terraform.
	
```terraform
provider "github" {
  owner = var.github_org
  token = var.github_token
}

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
```

The Flux provider needs to be configured both with Git and Kubernetes credentials. 

```terraform
provider "flux" {
  kubernetes = {
    host                   = kind_cluster.this.endpoint
    client_certificate     = kind_cluster.this.client_certificate
    client_key             = kind_cluster.this.client_key
    cluster_ca_certificate = kind_cluster.this.cluster_ca_certificate
  }
  git = {
    url = "ssh://git@github.com/${var.github_org}/${var.github_repository}.git"
    ssh = {
      username    = "git"
      private_key = tls_private_key.flux.private_key_pem
    }
  }
}

resource "flux_bootstrap_git" "this" {
  depends_on = [github_repository_deploy_key.this]

  path = "clusters/my-cluster"
}
```

Apply the Terraform, remember to include values for the varaibles.

```hcl
terraform apply -var "github_org=<username or org>" -var "github_token=<token>" -var "github_repository=fleet-infra"
```

When Terraform apply has completed a Kind cluster will exist with Flux installed and configured in it.
