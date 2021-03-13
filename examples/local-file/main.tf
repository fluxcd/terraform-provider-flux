terraform {
  required_version = ">= 0.14"

  required_providers {
    aws = {
      # https://github.com/terraform-providers/terraform-provider-aws/releases
      source  = "hashicorp/aws"
      version = ">= 3.31.0"
    }
    local = {
      # https://github.com/terraform-providers/terraform-provider-local/releases
      source  = "hashicorp/local"
      version = ">= 2.1.0"
    }
    flux = {
      # https://github.com/fluxcd/terraform-provider-flux/releases
      source  = "fluxcd/flux"
      version = ">= 0.0.13"
    }
    kubectl = {
      # https://github.com/gavinbunney/terraform-provider-kubectl/releases
      source  = "gavinbunney/kubectl"
      version = ">= 1.10.0"
    }
  }
}

data "flux_install" "main" {
  target_path = var.target_path
}

data "flux_sync" "main" {
  target_path = var.target_path
  url         = var.repository_url
  branch      = var.repository_branch
}

resource "local_file" "gotk_components" {
  content              = data.flux_install.main.content
  filename             = "${var.target_path}/flux-system/gotk-components.yaml"
  directory_permission = "0755"
  file_permission      = "0644"
}

resource "local_file" "gotk_sync" {
  content              = data.flux_sync.main.content
  filename             = "${path.module}/${var.target_path}/flux-system/gotk-sync.yaml"
  directory_permission = "0755"
  file_permission      = "0644"
}

variable "repository_branch" {
  default     = "main"
  description = "Name of the branch for git repository"
  type        = string
}

variable "repository_url" {
  default     = "ssh://git@bitbucket.org/ORG/REPO.git"
  description = "URL for the git repository"
  type        = string
}

variable "target_path" {
  default     = "flux/clusters/example"
  description = "Path in repository tree for Flux to find its manifests"
  type        = string
}
