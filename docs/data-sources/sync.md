---
page_title: "flux_sync Data Source - terraform-provider-flux"
subcategory: ""
description: |-
  flux_sync Returns the manifest for a gitsource configuration for flux
---

# Data Source `flux_sync`

`flux_sync` Returns the manifest for a gitsource configuration for flux



## Schema

### Required

- **target_path** (String, Required)
- **url** (String, Required)

### Optional

- **branch** (String, Optional)
- **id** (String, Optional) The ID of this resource.
- **interval** (Number, Optional)
- **name** (String, Optional)
- **namespace** (String, Optional)

### Read-only

- **content** (String, Read-only)
- **path** (String, Read-only)


