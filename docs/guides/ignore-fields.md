---
subcategory: ""
page_title: "Ignoring Manifest Changes"
description: |-
    Optionally ignore updates on manifest fields outside terraform.
---

# Ignoring Manifest Changes

Kubernetes will mutate the applied manifestsOnce fluxcd is installed on the first apply, Kubernetes will mutate the applied manifests adding fields to the spec and metadata sections. 

If the `kubectl` provider is used as-is, the plans will always mark the `install` and `sync` manifests to update in place due to a mismatch on the yaml manifest. So on every plan you will get something like this:

```shell
Terraform will perform the following actions:

  # kubectl_manifest.install["apps/v1/deployment/flux-system/helm-controller"] will be updated in-place
~ resource "kubectl_manifest" "install" {
        id                      = "/apis/apps/v1/namespaces/flux-system/deployments/helm-controller"
        name                    = "helm-controller"
      ~ yaml_incluster          = (sensitive value)
        # (14 unchanged attributes hidden)
    }

  # kubectl_manifest.install["apps/v1/deployment/flux-system/source-controller"] will be updated in-place
~ resource "kubectl_manifest" "install" {
        id                      = "/apis/apps/v1/namespaces/flux-system/deployments/source-controller"
        name                    = "source-controller"
      ~ yaml_incluster          = (sensitive value)
        # (14 unchanged attributes hidden)
    }

  # kubectl_manifest.sync["source.toolkit.fluxcd.io/v1beta2/gitrepository/flux-system/flux-system"] will be updated in-place
~ resource "kubectl_manifest" "sync" {
        id                      = "/apis/source.toolkit.fluxcd.io/v1beta2/namespaces/flux-system/gitrepositorys/flux-system"
        name                    = "flux-system"
      ~ yaml_incluster          = (sensitive value)
        # (14 unchanged attributes hidden)
    }
```

The fix to remove this issue is to ignore the automated fields for changes. If the whole spec and annotations are ignored we are making sure that the flux data resource.

```terraform
resource "kubectl_manifest" "install" {
  for_each   = { for v in local.install : lower(join("/", compact([v.data.apiVersion, v.data.kind, lookup(v.data.metadata, "namespace", ""), v.data.metadata.name]))) => v.content }
  depends_on = [kubernetes_namespace.flux_system]
  yaml_body  = each.value
  ignore_fields = ["metadata.annotations", "spec"]
}

resource "kubectl_manifest" "sync" {
  for_each   = { for v in local.sync : lower(join("/", compact([v.data.apiVersion, v.data.kind, lookup(v.data.metadata, "namespace", ""), v.data.metadata.name]))) => v.content }
  depends_on = [kubernetes_namespace.flux_system]
  yaml_body  = each.value
  ignore_fields = ["metadata.annotations", "spec"]
}
```