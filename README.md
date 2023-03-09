# Terraform Provider Flux

This is the Terraform provider for Flux v2. The provider allows you to install Flux on Kubernetes and configure it to reconcile the cluster state from a Git repository.

## Get Started

Below is an example for how to bootstrap a Kubernetes cluster with flux. Refer to [registry.terraform.io](https://registry.terraform.io/providers/fluxcd/flux/latest) for detailed configuration documentation.

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
