# Terraform provider flux

This is the Terraform provider for Flux v2. The provider allows you to install Flux on Kubernetes and configure it to reconcile the cluster state from a Git repository.

## Get started

Flux can be bootstrapped using Terraform with a single Terraform resource. The provider needs to be configured with Kubernetes credentials in the same way that the Kubernetes and Helm Terraform providers are configured.
The resource is configured with the repository URL and git credentials. When applied the necessary Kubernetes manifests will be committed to the Git repository and applied to the Kubernetes cluster. Flux will then be 
verified that it can synchronize from the configured Git repository.

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
