---
page_title: "flux_install Data Source - terraform-provider-flux"
subcategory: ""
description: |-
  flux_install Returns the manifest to install flux
---

# Data Source `flux_install`

`flux_install` Returns the manifest to install flux



## Schema

### Required

- **target_path** (String, Required)

### Optional

- **arch** (String, Optional)
- **base_url** (String, Optional)
- **components** (Set of String, Optional)
- **id** (String, Optional) The ID of this resource.
- **image_pull_secrets** (String, Optional)
- **log_level** (String, Optional)
- **namespace** (String, Optional)
- **network_policy** (Boolean, Optional)
- **registry** (String, Optional)
- **timeout** (Number, Optional)
- **version** (String, Optional)
- **watch_all_namespaces** (Boolean, Optional)

### Read-only

- **content** (String, Read-only)
- **path** (String, Read-only)


