---
subcategory: ""
page_title: "Reconcile 3rd-party Git sources"
description: |-
    A guide for setting up reconciliation of 3rd-party Git sources after Flux bootstrap.
---

# Reconcile 3rd-party Git sources

To generate `GitRepository` and `Kustomization` objects for reconciling
other repositories besides the bootstrap one,
you can use the [flux2-sync Helm chart](https://artifacthub.io/packages/helm/fluxcd-community/flux2-sync)
and the Helm Terraform Provider.

Example:

```terraform
terraform {
  required_providers {
    helm = {
      source  = "hashicorp/helm"
    }
    kind = {
      source  = "tehcyx/kind"
    }
  }
}

provider "kind" {}

resource "kind_cluster" "this" {
  name = "flux-e2e"
}

provider "helm" {
  kubernetes {
    host                   = kind_cluster.this.endpoint
    client_certificate     = kind_cluster.this.client_certificate
    client_key             = kind_cluster.this.client_key
    cluster_ca_certificate = kind_cluster.this.cluster_ca_certificate
  }
}

resource "helm_release" "my-app-sync" {
  repository       = "https://fluxcd-community.github.io/helm-charts"
  chart            = "flux2-sync"
  name             = "my-app-sync"
  namespace        = "flux-system"
  values = [
    file("${path.module}/my-app-sync-values.yaml")
  ]
}
```
