---
subcategory: ""
page_title: "Bootstrap with GitLab"
description: |-
	Install Flux and synchronize with GitLab.
---

# Bootstrap with GitLab

This guide will walk through how to install Flux into a Kubernetes cluster and configure it to synchronize from a GitLab repository. Begin with creating a GitLab repository, in this guide it will be named `fleet-infra`, which will be used by Flux.
Generate a [personal access token](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html) (PAT) that grants complete read/write access to the GitLab API., make sure to copy the generated token.

It is a good idea to use variables so that you do not accidentally store GitLab credentials in your Terraform configuration.

```terraform
variable "gitlab_token" {
  sensitive = true
  type      = string
}

variable "gitlab_group" {
  type = string
}

variable "gitlab_project" {
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
      version = ">=0.25.3"
    }
    kind = {
      source  = "tehcyx/kind"
      version = ">=0.0.16"
    }
    gitlab = {
      source  = "gitlabhq/gitlab"
      version = ">=15.10.0"
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

The GitLab repository is created separatly so a datasource is used to get a reference to the repository. Creating GitLab repositories with Terraform is generally not a good idea as they could easily be removed. Additionally it is not possible to use the same repository for multiple environments if the repository is created with Terraform.
	
```terraform
provider "gitlab" {
  token = var.gitlab_token
}

resource "tls_private_key" "flux" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

data "gitlab_project" "this" {
  path_with_namespace = "${var.gitlab_group}/${var.gitlab_project}"
}

resource "gitlab_deploy_key" "this" {
  project  = data.gitlab_project.this.id
  title    = "Flux"
  key      = tls_private_key.flux.public_key_openssh
  can_push = true
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
    url = "ssh://git@gitlab.com/${data.gitlab_project.this.path_with_namespace}.git"
    ssh = {
      username    = "git"
      private_key = tls_private_key.flux.private_key_pem
    }
  }
}

resource "flux_bootstrap_git" "this" {
  depends_on = [gitlab_deploy_key.this]

  path = "clusters/my-cluster"
}
```

Apply the Terraform, remember to include values for the varaibles.

```hcl
terraform apply -var "gitlab_group=<group>" -var "gitlab_token=<token>" -var "gitlab_repository=fleet-infra"
```

When Terraform apply has completed a Kind cluster will exist with Flux installed and configured in it.
