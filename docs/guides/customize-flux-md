---
subcategory: ""
page_title: "Customize Flux"
description: |-
  Customizing Flux past the exposed parameters.
---

# Customize Flux

The Flux datasources expose a set of parameters that can configure the controller deployment. These parameters are identical to the ones
exposed by the `flux bootstrap` CLI command. There may be situations where the exposed parameters are not enough and additional configuration
has to be done. This could be changing the resource requests or limits for a controller, adding annotations or labels, or modifying container settings.
When deploying Flux with the CLI the recommended solution is to [modify the Kustomization file](https://fluxcd.io/docs/installation/#customize-flux-manifests).
This is also the recommended way when deploying with Terraform, but it requires some extra considerations as changes made manually to
the repository after deployment would be overridden the next time the Terraform is applied.

This guide assumes that you have setup Flux with Terraform already. Follow the [GitHub guide](./github) for a quick example to get a Kubernetes cluster with Flux installed in it.
Customizing Flux with Terraform requires you to create your own Kustomization file instead of using the one given by the provider.

The following path file will set PSP rules for all of the Flux deployments.
```terraform
apiVersion: apps/v1
kind: Deployment
metadata:
  name: all-flux-components
spec:
  template:
    metadata:
      annotations:
        # Required by Kubernetes node autoscaler
        cluster-autoscaler.kubernetes.io/safe-to-evict: "true"
    spec:
      securityContext:
        runAsUser: 10000
        fsGroup: 1337
      containers:
        - name: manager
          securityContext:
            readOnlyRootFilesystem: true
            allowPrivilegeEscalation: false
            runAsNonRoot: true
            capabilities:
              drop:
                - ALL
```

The path file can then be used in the Kustomization file. It is important that the resources list contains the `gotk-components.yaml` and `gotk-sync.yaml` files as otherwise the Flux
manifests will not be included.
```terraform
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- gotk-components.yaml
- gotk-sync.yaml
patches:
  - path: psp-patch.yaml
    target:
      kind: Deployment
```

Change the Terraform so that it will commit the local Kustomization file instead of the one given by the provider. Make sure that the patch file is also committed to the same repository.
```terraform
terraform {
  required_version = ">= 0.13"

  required_providers {
    github = {
      source = "integrations/github"
      version = "4.5.2"
    }
  }
}

resource "github_repository_file" "kustomize" {
  repository = github_repository.main.name
  file       = data.flux_sync.main.kustomize_path
  content    = file("${path.module}/kustomization-override.yaml")
  branch     = var.branch
}

resource "github_repository_file" "psp_patch" {
  repository = github_repository.main.name
  file       = "${dirname(data.flux_sync.main.kustomize_path)}/psp-patch.yaml"
  content    = file("${path.module}/psp-patch.yaml")
  branch     = var.branch
}
```
