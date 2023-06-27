# Terraform Provider Flux

[![tests](https://github.com/fluxcd/terraform-provider-flux/workflows/tests/badge.svg)](https://github.com/fluxcd/terraform-provider-flux/actions)
[![report](https://goreportcard.com/badge/github.com/fluxcd/terraform-provider-flux)](https://goreportcard.com/report/github.com/fluxcd/terraform-provider-flux)
[![license](https://img.shields.io/github/license/fluxcd/terraform-provider-flux.svg)](https://github.com/fluxcd/terraform-provider-flux/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/fluxcd/terraform-provider-flux/all.svg)](https://github.com/fluxcd/terraform-provider-flux/releases)

This is the Terraform provider for Flux v2. The provider allows you to install Flux on Kubernetes
and configure it to reconcile the cluster state from a Git repository.

## Get Started

Below is an example for how to bootstrap a Kubernetes cluster with Flux.
Refer to [registry.terraform.io](https://registry.terraform.io/providers/fluxcd/flux/latest)
for detailed configuration documentation.

```hcl
provider "flux" {
  kubernetes = {
    config_path = "~/.kube/config"
  }
  git = {
    url  = var.repository_ssh_url
    ssh = {
      username    = "git"
      private_key = var.private_key_pem
    }
  }
}

resource "flux_bootstrap_git" "this" {
  path = "clusters/my-cluster"
}
```

## Guides

* [Customize Flux configuration](https://registry.terraform.io/providers/fluxcd/flux/latest/docs/resources/bootstrap_git#customizing-flux)
* [Bootstrap with GitHub](https://registry.terraform.io/providers/fluxcd/flux/latest/docs/guides/github)
* [Bootstrap with GitLab](https://registry.terraform.io/providers/fluxcd/flux/latest/docs/guides/gitlab)
