terraform {
  required_version = ">=1.1.5"

  required_providers {
    flux = {
      source  = "fluxcd/flux"
      version = ">= 1.0.0-rc.1"
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
