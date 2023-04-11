---
page_title: "Multi Environment Considerations for flux_install"
subcategory: "Deprecated"
description: |-
    Considerations when deploying multiple environments.
---

# Multi Environment Considerations

Usually when deploying multiple environments the same Terraform HCL is used, using different backends to store the state.
This introduces an issue with the simple examples as they create the git repository for you. The first environment would
deploy properly, but the second would fail as it attempts to create an identical repository. Generally you would want
to share the same repository for the Flux deployments in the different environments. The solution to the problem is
to manually create the repository and then use a datasource instead of a resource.

```terraform
terraform {
  required_version = ">= 1.1.5"

  required_providers {
    github = {
      source  = "integrations/github"
      version = "4.5.2"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "3.1.0"
    }
  }
}


data "github_repository" "main" {
  name = var.repository_name
}

resource "tls_private_key" "main" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

resource "github_repository_deploy_key" "main" {
  title      = "flux-${var.environment}"
  repository = data.github_repository.main.name
  key        = tls_private_key.main.public_key_openssh
  read_only  = true
}

resource "github_repository_file" "install" {
  repository = data.github_repository.main.name
  file       = data.flux_install.main.path
  content    = data.flux_install.main.content
  branch     = var.branch
}

resource "github_repository_file" "sync" {
  repository = data.github_repository.main.name
  file       = data.flux_sync.main.path
  content    = data.flux_sync.main.content
  branch     = var.branch
}

resource "github_repository_file" "kustomize" {
  repository = data.github_repository.main.name
  file       = data.flux_sync.main.kustomize_path
  content    = data.flux_sync.main.kustomize_content
  branch     = var.branch
}
```

Manually creating repositories instead of letting Terraform track them also removes the risk of accidentally deleting
a repository. Bootstrapping Flux will require Terraform to commit files to the repository, and allow Terraform to
overwrite files if you change them. It is likely that other files have been committed to the same repository that
Terraform has not created, in that case it would not be optimal if Terraform removed the repository when
destroying other resources.
