---
subcategory: ""
page_title: "Breaking changes to the provider configuration"
description: |-
    Guide for how to deault with the breaking changes made to the provider configuration.
---

# Breaking changes to the provider configuration

Change have been made to the provider configuration to move any Git credential configuration from the individual resources to the provider.
This is to simplify the process of adding more resources without requiring end users to configure the same Git crendentials in each resource.
As part of the breaking change the existing Kuberentes configuration has been moved inside of a single nested attributed, which groups relevant attributes together.

Before the breaking change simple bootstrap resource configuration could look like the following.

```hcl
provider "flux" {
  config_path = "~/.kube/config"
}

resource "flux_bootstrap_git" "this" {
  url = "https://github.com/username/repository.git"
  http = {
    username = "git"
    password = "password"
  }
  path = "staging-cluster"
}
```

The same configuration after the breaking change would require moving some attributes to the provider.

```hcl
provider "flux" {
  kubernetes = {
    config_path = "~/.kube/config"
  }
  git = {
    url = "https://github.com/username/repository.git"
    http = {
      username = "git"
      password = "password"
    }
  }
}

resource "flux_bootstrap_git" "this" {
  path = "staging-cluster"
}
```

These changes should not cause any changes in a plan as long as other attributes have not been changed.
