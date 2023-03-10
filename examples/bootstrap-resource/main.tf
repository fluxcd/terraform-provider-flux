terraform {
  required_version = ">= 1.1.5"
  required_providers {
    flux = {
      source  = "registry.terraform.io/fluxcd/flux"
      version = ">= 0.25.0"
    }
  }
}

provider "flux" {
  kubernetes = {
    config_path = "~/.kube/config"
  }
  git = {
    url = "https://github.com/${var.username}/fleet-infra"
    http = {
      username = var.username
      password = var.password
    }
  }
}

resource "flux_bootstrap_git" "this" {}
