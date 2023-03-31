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
