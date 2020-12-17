---
page_title: "flux_sync Data Source - terraform-provider-flux"
subcategory: ""
description: |-
  flux_sync can be used to generate manifests for reconciling the specified repository path on the cluster.
---

# Data Source `flux_sync`

`flux_sync` can be used to generate manifests for reconciling the specified repository path on the cluster.

## Example Usage

```terraform
variable "target_path" {
  type = string
}

variable "clone_url" {
  type = string
}

data "flux_sync" "main" {
  target_path = var.target_path
  url         = var.clone_url
}
```

## Schema

### Required

- **target_path** (String, Required) Relative path to the Git repository root where the sync manifests are committed.
- **url** (String, Required) Git repository clone url.

### Optional

- **branch** (String, Optional) Default branch to sync from. Defaults to `main`.
- **id** (String, Optional) The ID of this resource.
- **interval** (Number, Optional) Sync interval in minutes. Defaults to `1`.
- **name** (String, Optional) The kubernetes resources name Defaults to `flux-system`.
- **namespace** (String, Optional) The namespace scope for this operation. Defaults to `flux-system`.

### Read-only

- **content** (String, Read-only) Manifests in multi-doc yaml format.
- **kustomize_content** (String, Read-only) Kustomize yaml document.
- **kustomize_path** (String, Read-only) Expected path of kustomize content in git repository.
- **path** (String, Read-only) Expected path of content in git repository.


