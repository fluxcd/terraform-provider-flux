terraform {
  required_version = ">= 1.1.5"
  required_providers {
    flux = {
      source = "registry.terraform.io/fluxcd/flux"
      version = "0.0.0-dev"
    }
  }
}

provider "flux" {
  config_path = "~/.kube/config"
}

resource "flux_bootstrap_git" "this" {
  url = "https://github.com/${var.username}/fleet-infra"
  http = {
    username = var.username
    password = var.password
  }
}
