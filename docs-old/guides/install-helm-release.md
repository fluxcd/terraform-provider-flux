---
subcategory: ""
page_title: "Install Only with Helm Chart"
description: |-
  A guide for installing Flux with Terraform without any bootstrap.
---

# Install Only with Helm Chart

Using the [Flux Helm chart](https://github.com/fluxcd-community/helm-charts/tree/main/charts/flux2) is a better option when Flux needs to be installed without any bootstrap configuration.
The Helm Terraform provider can be used to install the chart in a Kubernetes cluster. Custom resource definitions will be applied along with all of the Flux components.

Define a Kind cluster and pass the cluster configuration to the Helm provider along with a `helm_release` resource that refers to the Flux Helm chart.

```terraform
terraform {
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = ">=2.9.0"
    }
    kind = {
      source  = "tehcyx/kind"
      version = ">=0.0.16"
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

resource "helm_release" "this" {
  repository       = "https://fluxcd-community.github.io/helm-charts"
  chart            = "flux2"
  name             = "flux2"
  namespace        = "flux-system"
  create_namespace = true
}
```

After applying the example Terraform configuration, a Kind cluster should exist with Flux installed in the `flux-system` namespace.
