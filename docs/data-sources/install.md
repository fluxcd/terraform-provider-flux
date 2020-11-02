---
page_title: "flux_install Data Source - terraform-provider-flux"
subcategory: ""
description: |-
  flux_install can be used to get install manifests for Flux.
---

# Data Source `flux_install`

`flux_install` can be used to get install manifests for Flux.

## Example Usage

```terraform
variable "target_path" {
  type = string
}

data "flux_install" "main" {
  target_path = var.target_path
}
```

## Schema

### Required

- **target_path** (String, Required) Path to use when computing manifest file path.

### Optional

- **arch** (String, Optional) Cluster architecture for toolkit container images. Defaults to `amd64`.
- **components** (Set of String, Optional) Toolkit components to include in the install manifests.
- **id** (String, Optional) The ID of this resource.
- **image_pull_secrets** (String, Optional) Kubernetes secret name used for pulling the toolkit images from a private registry. Defaults to ``.
- **log_level** (String, Optional) Log level for toolkit components. Defaults to `info`.
- **namespace** (String, Optional) The namespace scope for install manifests. Defaults to `flux-system`.
- **network_policy** (Boolean, Optional) Deny ingress access to the toolkit controllers from other namespaces using network policies. Defaults to `true`.
- **registry** (String, Optional) Container registry where the toolkit images are published. Defaults to `ghcr.io/fluxcd`.
- **version** (String, Optional) Toolkit version. Defaults to `latest`.
- **watch_all_namespaces** (Boolean, Optional) If true watch for custom resources in all namespaces. Defaults to `true`.

### Read-only

- **content** (String, Read-only) Manifests in multi-doc yaml format.
- **path** (String, Read-only) Expected path of content in git repository.


