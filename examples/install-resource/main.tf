terraform {
  required_providers {
    flux = {
      source = "registry.terraform.io/fluxcd/flux"
      version = "0.0.0-dev"
    }
  }
}

provider "flux" {
}

resource "flux_install" "this" {
  components           = [
           "helm-controller",
           "kustomize-controller",
           #"notification-controller",
           "source-controller",
        ]
}
