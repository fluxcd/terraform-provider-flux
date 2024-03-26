# Terraform Provider Flux

[![tests](https://github.com/fluxcd/terraform-provider-flux/workflows/tests/badge.svg)](https://github.com/fluxcd/terraform-provider-flux/actions)
[![report](https://goreportcard.com/badge/github.com/fluxcd/terraform-provider-flux)](https://goreportcard.com/report/github.com/fluxcd/terraform-provider-flux)
[![license](https://img.shields.io/github/license/fluxcd/terraform-provider-flux.svg)](https://github.com/fluxcd/terraform-provider-flux/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/fluxcd/terraform-provider-flux/all.svg)](https://github.com/fluxcd/terraform-provider-flux/releases)

This is the Terraform provider for Flux v2. The provider allows you to install Flux on Kubernetes and configure it to reconcile the cluster state from a Git repository.

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

The following guides are available to help you use the provider:

- [Configuration using a Github repository via SSH](examples/github-via-ssh)
- [Configuration using a Github repository via SSH and GPG](examples/github-via-ssh-with-gpg)
- [Configuration using a Github repository via SSH with flux customizations](examples/github-with-customizations)
- [Configuration using a Github repository via SSH and GPG with inline flux customizations](examples/github-with-inline-customizations)
- [Configuration using a Gitlab repository via SSH](examples/gitlab-via-ssh)
- [Configuration using a Gitlab repository via SSH and GPG](examples/gitlab-via-ssh-with-gpg)
- [Configuration using a Helm Release and not the flux_bootstrap_git resource](examples/helm-install) **

** This is the recommended approach if you do not want to perform initial flux bootstrapping.
