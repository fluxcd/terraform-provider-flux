# Changelog

All notable changes to this project are documented in this file.

## 0.0.10

The `arch` parameters has been removed from the `flux_install` resource
as multi-arch images are now published under the same tag.

Improvements:
* Update flux to v0.6.0
  [#77](https://github.com/fluxcd/terraform-provider-flux/pull/77)
* Add git implementation to sync datasource
  [#76](https://github.com/fluxcd/terraform-provider-flux/pull/76)
