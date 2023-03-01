terraform {
  required_version = ">= 1.1.5"
  required_providers {
    flux = {
      source  = "registry.terraform.io/fluxcd/flux"
      version = "0.0.0-dev"
    }
  }
}

provider "flux" {
}

resource "flux_install" "this" {
}
