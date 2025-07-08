# Changelog

All notable changes to this project are documented in this file.

## 1.6.4

**Release date:** 2025-07-08

This release includes flux2 [v2.6.4](https://github.com/fluxcd/flux2/releases/tag/v2.6.4).

## 1.6.3

**Release date:** 2025-06-27

This release includes flux2 [v2.6.3](https://github.com/fluxcd/flux2/releases/tag/v2.6.3).

## 1.6.2

**Release date:** 2025-06-16

This release includes flux2 [v2.6.2](https://github.com/fluxcd/flux2/releases/tag/v2.6.2).

## 1.6.1

**Release date:** 2025-06-02

This release includes flux2 [v2.6.1](https://github.com/fluxcd/flux2/releases/tag/v2.6.1).

## 1.6.0

**Release date:** 2025-05-29

This release includes flux2 [v2.6.0](https://github.com/fluxcd/flux2/releases/tag/v2.6.0).

In addition, the Kubernetes dependencies have been updated to v1.33.0
and the provider is now built with Go 1.24.

Improvements:
- Update to Kubernetes 1.33.0 and Go 1.24.0
  [#757](https://github.com/fluxcd/terraform-provider-flux/pull/757)
- Update Flux to v2.6.0
  [#760](https://github.com/fluxcd/terraform-provider-flux/pull/760)

## 1.5.1

**Release date:** 2025-02-25

This release includes flux2 [v2.5.1](https://github.com/fluxcd/flux2/releases/tag/v2.5.1).

Improvements:
- Update to flux2 v2.5.1
  [#749](https://github.com/fluxcd/terraform-provider-flux/pull/749)

## 1.5.0

**Release date:** 2025-02-20

This release includes flux2 [v2.5.0](https://github.com/fluxcd/flux2/releases/tag/v2.5.0).

In addition, the Kubernetes dependencies have been updated to v1.32.1.

Improvements:
- Various dependency updates
  [#724](https://github.com/fluxcd/terraform-provider-flux/pull/724)
  [#725](https://github.com/fluxcd/terraform-provider-flux/pull/725)
  [#726](https://github.com/fluxcd/terraform-provider-flux/pull/726)
  [#727](https://github.com/fluxcd/terraform-provider-flux/pull/727)
  [#729](https://github.com/fluxcd/terraform-provider-flux/pull/729)
  [#739](https://github.com/fluxcd/terraform-provider-flux/pull/739)
  [#741](https://github.com/fluxcd/terraform-provider-flux/pull/741)
  [#742](https://github.com/fluxcd/terraform-provider-flux/pull/742)

## 1.4.0

**Release date:** 2024-09-30

This release includes flux2 [v2.4.0](https://github.com/fluxcd/flux2/releases/tag/v2.4.0).

The client-go auth plugins are now imported, which allows using auth providers
like OIDC, GCP, Azure, etc., for connecting to the cluster.

In addition, the Kubernetes dependencies have been updated to v1.31.1 and the
provider is now built with Go 1.23.

Improvements:
- Build with Go 1.23
  [#710](https://github.com/fluxcd/terraform-provider-flux/pull/710)
- Import client-go auth plugin to fix oidc auth issue
  [#702](https://github.com/fluxcd/terraform-provider-flux/pull/702)
- Various dependency updates
  [#691](https://github.com/fluxcd/terraform-provider-flux/pull/691)
  [#698](https://github.com/fluxcd/terraform-provider-flux/pull/698)
  [#699](https://github.com/fluxcd/terraform-provider-flux/pull/699)
  [#701](https://github.com/fluxcd/terraform-provider-flux/pull/701)
  [#707](https://github.com/fluxcd/terraform-provider-flux/pull/707)
  [#711](https://github.com/fluxcd/terraform-provider-flux/pull/711)
  [#715](https://github.com/fluxcd/terraform-provider-flux/pull/715)
  [#718](https://github.com/fluxcd/terraform-provider-flux/pull/718)
  [#719](https://github.com/fluxcd/terraform-provider-flux/pull/719)

## 1.3.0

**Release date:** 2024-05-13

This release includes flux2 [v2.3.0](https://github.com/fluxcd/flux2/releases/tag/v2.3.0).

The provider has undergone a major refactoring and now supports air-gapped bootstrap,
drift detection and correction for Flux components, and the ability to upgrade and
restore the Flux controllers in-cluster.

New configuration options in `flux_bootstrap_git`:

- `delete_git_manifests` (Boolean) Delete manifests from git repository. Defaults to `true`.
- `embedded_manifests` (Boolean) When enabled, the Flux manifests will be extracted from the provider binary instead of being downloaded from GitHub.com. Defaults to `false`.
- `registry_credentials` (String) Container registry credentials in the format `user:password`.

Starting with this release, the provider is fully compatible with OpenTofu.

The [provider documentation](https://github.com/fluxcd/terraform-provider-flux?tab=readme-ov-file#guides)
has been updated with examples and detailed usage instructions.

The deprecated resources `flux_install` and `flux_sync` have been removed.

Improvements:
- Update Flux to v2.3.0
  [#689](https://github.com/fluxcd/terraform-provider-flux/pull/689)
- Add registry credential support to bootstrap resource
  [#688](https://github.com/fluxcd/terraform-provider-flux/pull/688)
- Improve readiness diagnotics messages
  [#680](https://github.com/fluxcd/terraform-provider-flux/pull/680)
- Add `hostkey_algos` to the `git.ssh` schema
  [#679](https://github.com/fluxcd/terraform-provider-flux/pull/679)
- Update terraform plugin framework to v1.8.0
  [#674](https://github.com/fluxcd/terraform-provider-flux/pull/674)
- Update dependencies to Kubernetes 1.30
  [#673](https://github.com/fluxcd/terraform-provider-flux/pull/673)
- Update flux update GH action to regen docs
  [#671](https://github.com/fluxcd/terraform-provider-flux/pull/671)
- Set `embedded_manifest` to true and repo visibility to private
  [#666](https://github.com/fluxcd/terraform-provider-flux/pull/666)
- Implement drift detection and correction for cluster state
  [#661](https://github.com/fluxcd/terraform-provider-flux/pull/661)
- Provide an option not to delete the namespace Flux is installed into
  [#657](https://github.com/fluxcd/terraform-provider-flux/pull/657)
- Add optional git manifest delete
  [#650](https://github.com/fluxcd/terraform-provider-flux/pull/650)
- Using the flux2-sync helm chart in the examples
  [#636](https://github.com/fluxcd/terraform-provider-flux/pull/636)
- Removing flux_install and flux_sync data sources
  [#630](https://github.com/fluxcd/terraform-provider-flux/pull/630)
- Updating examples to include repository creation
  [#621](https://github.com/fluxcd/terraform-provider-flux/pull/621)
- Updated examples, simplified documentation and adding pre-commit to CI
  [#616](https://github.com/fluxcd/terraform-provider-flux/pull/616)

## 1.2.3

**Release date:** 2024-02-05

This release includes flux2 [v2.2.3](https://github.com/fluxcd/flux2/releases/tag/v2.2.3).

## 1.2.2

**Release date:** 2023-12-19

This release includes flux2 [v2.2.2](https://github.com/fluxcd/flux2/releases/tag/v2.2.2).

## 1.2.1

**Release date:** 2023-12-15

This release includes flux2 [v2.2.1](https://github.com/fluxcd/flux2/releases/tag/v2.2.1).

## 1.2.0

**Release date:** 2023-12-12

This release includes flux2 [v2.2.0](https://github.com/fluxcd/flux2/releases/tag/v2.2.0).

Improvements:
- docs: Provider can deploy to pre-existing namespace
  [#577](https://github.com/fluxcd/terraform-provider-flux/pull/577)
- docs: Fix namespace typo in install-helm-release.md
  [#576](https://github.com/fluxcd/terraform-provider-flux/pull/576)
- Address various issues throughout code base
  [#565](https://github.com/fluxcd/terraform-provider-flux/pull/565)
- Support air-gapped, offline, or otherwise customized install manifests
  [#503](https://github.com/fluxcd/terraform-provider-flux/pull/503)
- Dependency updates
  [#579](https://github.com/fluxcd/terraform-provider-flux/pull/579)
  [#567](https://github.com/fluxcd/terraform-provider-flux/pull/567)
  [#582](https://github.com/fluxcd/terraform-provider-flux/pull/582)

Fixes:
- Fix signing commits with GPG key
  [#572](https://github.com/fluxcd/terraform-provider-flux/pull/572)

## 1.1.2

**Release date:** 2023-10-12

This release includes flux2 [v2.1.2](https://github.com/fluxcd/flux2/releases/tag/v2.1.2).

## 1.1.1

**Release date:** 2023-09-19

This release includes flux2 [v2.1.1](https://github.com/fluxcd/flux2/releases/tag/v2.1.1).

## 1.1.0

**Release date:** 2023-08-24

This release includes flux2 [v2.1.0](https://github.com/fluxcd/flux2/releases/tag/v2.1.0).

Improvements:
- Update dependencies
  [#540](https://github.com/fluxcd/terraform-provider-flux/pull/540),
  [#533](https://github.com/fluxcd/terraform-provider-flux/pull/533),
  [#525](https://github.com/fluxcd/terraform-provider-flux/pull/525),
  [#524](https://github.com/fluxcd/terraform-provider-flux/pull/524)
- Fix typo
  [#522](https://github.com/fluxcd/terraform-provider-flux/pull/522)

## 1.0.1

**Release date:** 2023-07-11

This release includes flux2 [v2.0.1](https://github.com/fluxcd/flux2/releases/tag/v2.0.1).

Improvements:
- Update dependencies
  [#519](https://github.com/fluxcd/terraform-provider-flux/pull/519)
- Add test for missing configuration
  [#515](https://github.com/fluxcd/terraform-provider-flux/pull/515)

## 1.0.0

**Release date:** 2023-07-05

This is the first stable release of the Terraform provider for Flux. From now on,
this provider follows the [Flux 2 release cadence and support pledge](https://fluxcd.io/flux/releases/).

Starting with this version, the build, release and provenance portions of the
Flux project supply chain [provisionally meet SLSA Build Level 3](https://fluxcd.io/flux/security/slsa-assessment/).

This release adds support for using exec plugins to authenticate the kubernetes
client used in the bootstrap git resource.

Improvements:
- Update flux dependencies and init logger
  [#513](https://github.com/fluxcd/terraform-provider-flux/pull/513)
- Update docs for v1
  [#510](https://github.com/fluxcd/terraform-provider-flux/pull/510)
- Declaratively define (and sync) labels
  [#507](https://github.com/fluxcd/terraform-provider-flux/pull/507)
- Add Kubernetes client auth exec config support
  [#506](https://github.com/fluxcd/terraform-provider-flux/pull/506)
- Align go.mod version with Kubernetes (Go 1.20)
  [#505](https://github.com/fluxcd/terraform-provider-flux/pull/505)
- Add SLSA3 generator to release workflow
  [#504](https://github.com/fluxcd/terraform-provider-flux/pull/504)
- Update dependencies
  [#502](https://github.com/fluxcd/terraform-provider-flux/pull/502)
  [#497](https://github.com/fluxcd/terraform-provider-flux/pull/497)
  [#496](https://github.com/fluxcd/terraform-provider-flux/pull/496)

Fixes:
- Fix panic due to missing configuration
  [#509](https://github.com/fluxcd/terraform-provider-flux/pull/509)

## 1.0.0-rc.5

**Release date:** 2023-06-01

This release includes flux2 [v2.0.0-rc.5](https://github.com/fluxcd/flux2/releases/tag/v2.0.0-rc.5).

Flux `v2.0.0-rc.5` addresses a regression that was introduced in `v2.0.0-rc.4`.
This regression caused a disruption in the compatibility with Git servers
utilizing v2 of the wire protocol, such as Azure Devops and AWS CodeCommit.

## 1.0.0-rc.4

**Release date:** 2023-05-29

This release includes flux2 [v2.0.0-rc.4](https://github.com/fluxcd/flux2/releases/tag/v2.0.0-rc.4).

## 1.0.0-rc.3

**Release date:** 2023-05-12

This release includes flux2 [v2.0.0-rc.3](https://github.com/fluxcd/flux2/releases/tag/v2.0.0-rc.3).

## 1.0.0-rc.2

**Release date:** 2023-05-10

This release includes flux2 [v2.0.0-rc.2](https://github.com/fluxcd/flux2/releases/tag/v2.0.0-rc.2).

This release includes the new attribute `disable_secret_creation` in the `flux_bootstrap_git` resource. This empowers users to override the git credentials used by Flux inside of the Kubernetes cluster.

## 1.0.0-rc.1

**Release date:** 2023-04-11

This release includes flux2 [v2.0.0-rc.1](https://github.com/fluxcd/flux2/releases/tag/v2.0.0-rc.1).

The datasources `flux_install` and `flux_sync` are deprecated, with a warning about their removal in the future. Users of the datasources should migrate to the `flux_bootstrap_git` resources using the [migration guide](https://registry.terraform.io/providers/fluxcd/flux/latest/docs/guides/migrating-to-resource).

This release also implements a timeout configuration for `flux_bootstrap_git` which enables retrying of update actions when multiple commits are pushed in parallel.

## 0.25.3

**Release date:** 2023-03-21

This prerelease includes flux2 [v0.41.2](https://github.com/fluxcd/flux2/releases/tag/v0.41.2).

## 0.25.2

**Release date:** 2023-03-20

This prerelease includes flux2 [v0.41.1](https://github.com/fluxcd/flux2/releases/tag/v0.41.1).

Fixes errors caused by passing computed values to certain Git configuration attributes in the provider by delaying the reading of the attributes.

## 0.25.1

**Release date:** 2023-03-10

This prerelease includes flux2 [v0.41.0](https://github.com/fluxcd/flux2/releases/tag/v0.41.0).

Fixes to components extra configuration in `flux_install` datasource which caused it to not be included in the generated manifests.

## 0.25.0

**Release date:** 2023-03-09

This prerelease includes flux2 [v0.41.0](https://github.com/fluxcd/flux2/releases/tag/v0.41.0).

This release contains a breaking configuration change. The git repository configuration including the credentials has been moved from the `flux_bootstrap_git` resource to the provider block. Detailed information can be found in the [documentation](./docs/guides/breaking-provider-configuration.md).

## 0.24.2

**Release date:** 2023-02-28

This prerelease includes flux2 [v0.40.2](https://github.com/fluxcd/flux2/releases/tag/v0.40.2).

## 0.24.1

**Release date:** 2023-02-23

This prerelease includes flux2 [v0.40.1](https://github.com/fluxcd/flux2/releases/tag/v0.40.1).

## 0.24.0

**Release date:** 2023-02-20

This prerelease includes flux2 [v0.40.0](https://github.com/fluxcd/flux2/releases/tag/v0.40.0).

This release contains a breaking change with the removal of the `git_implementation` attribute from the `flux_sync` data source. The libgit2 implementation has been removed from Flux as the `go-git` implementation supports all Git servers, including Azure DevOps and AWS CodeCommit.

Some minor fixes have been made to the new `flux_bootstrap_git` resource as users were experiencing issues setting custom components configuration.

## 0.23.0

**Release date:** 2023-02-02

This prerelease includes flux2 [v0.39.0](https://github.com/fluxcd/flux2/releases/tag/v0.39.0).

A new resource `flux_bootstrap_git` has been added as a replacement to the current two data source.

Bootstrapping Flux with Terraform has been a feature that was available early on in Fluxcd V2 development. With time new features in Flux and more requirements from end users have made the experience complex and error-prone.
A big reason is that the solution was built with a focus on using existing providers to manage the interaction with Git and Kubernetes. While it saved time early on it caused issues in the long run.
Flux has specific requirements in the order resources are applied and removed to work properly, which became very difficult to express with Terraform. In the worst case, Terraform would not be able to run at all due to the dependency complexity.

Over the last couple of months, a new provider resource has been developed to replace the old bootstrapping method. The resource implements all the functionality required to bootstrap Flux, removing the dependency on third-party Terraform providers.
The goal has been to replicate the features offered by the Flux CLI as close as possible while solving long-standing issues current users experience with the provider.

Currently customizing the Flux installation has issues as it only affects how Flux reconciles itself but now how the manifests are applied to Kubernetes. This results in issues for the end user as their cluster may block Pods lacking specific security settings.
This issue is no longer present in the new solution as the provider manages both the process of committing the customized configuration to Git and applying it to Kubernetes.

Another long-standing issue has been uninstalling Flux with Terraform. The old solution was for the most part luck based if the resources were removed in the right order and enough time was allowed for finalizers to be removed by the Flux controllers.
This rarely occurred causing clusters to be stuck in locked states. The new solution will always remove resources in the correct order, making sure that all finalizers are removed before Kubernetes considers the resource to be fully removed from Terraform.

The plan forward is to only work on developing the new Terraform resource, initially focusing on ironing out any early bugs. While the new resource is stable there may occur some breaking attribute changes to deal with unforeseen use cases. While having said that feedback
is needed to make the Terraform experience with Flux better, so please start evaluating the new Terraform resource. A migration guide is available which walks through the step-by-step process for how to migrate between the old and new solution without breaking an existing Flux installation.
When the attributes in the new resource are considered set, the old data sources will be deprecated, and eventually removed. Until then it is recommended to start looking at which features are missing in the new resource to cover specific use cases.

## 0.22.3

**Release date:** 2023-01-10

This prerelease includes flux2 [v0.38.3](https://github.com/fluxcd/flux2/releases/tag/v0.38.3).

## 0.22.2

**Release date:** 2022-12-22

This prerelease includes flux2 [v0.38.2](https://github.com/fluxcd/flux2/releases/tag/v0.38.2).

## 0.22.1

**Release date:** 2022-12-21

This prerelease includes flux2 [v0.38.1](https://github.com/fluxcd/flux2/releases/tag/v0.38.1).

## 0.22.0

**Release date:** 2022-12-21

This prerelease includes flux2 [v0.38.0](https://github.com/fluxcd/flux2/releases/tag/v0.38.0).

## 0.21.0

**Release date:** 2022-11-22

This prerelease includes flux2 [v0.37.0](https://github.com/fluxcd/flux2/releases/tag/v0.37.0).

## 0.20.0

**Release date:** 2022-10-24

This prerelease includes flux2 [v0.36.0](https://github.com/fluxcd/flux2/releases/tag/v0.36.0).

## 0.19.0

**Release date:** 2022-09-30

This prerelease includes flux2 [v0.35.0](https://github.com/fluxcd/flux2/releases/tag/v0.35.0).

## 0.18.0

**Release date:** 2022-09-12

This prerelease includes flux2 [v0.34.0](https://github.com/fluxcd/flux2/releases/tag/v0.34.0).

## 0.17.0

**Release date:** 2022-08-29

This prerelease includes flux2 [v0.33.0](https://github.com/fluxcd/flux2/releases/tag/v0.33.0).

## 0.16.0

**Release date:** 2022-08-11

This prerelease includes flux2 [v0.32.0](https://github.com/fluxcd/flux2/releases/tag/v0.32.0).

## 0.15.5

**Release date:** 2022-07-27

This prerelease includes flux2 [v0.31.5](https://github.com/fluxcd/flux2/releases/tag/v0.31.5).

## 0.15.4

**Release date:** 2022-07-26

This prerelease includes flux2 [v0.31.4](https://github.com/fluxcd/flux2/releases/tag/v0.31.4).

## 0.15.3

**Release date:** 2022-06-28

This prerelease includes flux2 [v0.31.3](https://github.com/fluxcd/flux2/releases/tag/v0.31.3).

## 0.15.2

**Release date:** 2022-06-28

This prerelease includes flux2 [v0.31.2](https://github.com/fluxcd/flux2/releases/tag/v0.31.2).

## 0.15.1

**Release date:** 2022-06-08

This prerelease includes flux2 [v0.31.1](https://github.com/fluxcd/flux2/releases/tag/v0.31.1).

## 0.15.0

**Release date:** 2022-06-06

This prerelease includes flux2 [v0.31.0](https://github.com/fluxcd/flux2/releases/tag/v0.31.0).

## 0.14.1

**Release date:** 2022-05-04

This prerelease includes flux2 [v0.30.2](https://github.com/fluxcd/flux2/releases/tag/v0.30.2).

## 0.14.0

**Release date:** 2022-05-04

This prerelease includes flux2 [v0.30.1](https://github.com/fluxcd/flux2/releases/tag/v0.30.1).

## 0.13.5

**Release date:** 2022-04-28

This prerelease includes flux2 [v0.29.5](https://github.com/fluxcd/flux2/releases/tag/v0.29.5).

## 0.13.4

**Release date:** 2022-04-26

This prerelease includes flux2 [v0.29.4](https://github.com/fluxcd/flux2/releases/tag/v0.29.4).

## 0.13.3

**Release date:** 2022-04-22

This prerelease includes flux2 [v0.29.3](https://github.com/fluxcd/flux2/releases/tag/v0.29.3).

## 0.13.2

**Release date:** 2022-04-21

This prerelease includes flux2 [v0.29.2](https://github.com/fluxcd/flux2/releases/tag/v0.29.2).

## 0.13.1

**Release date:** 2022-04-20

This prerelease includes flux2 [v0.29.1](https://github.com/fluxcd/flux2/releases/tag/v0.29.1).

## 0.13.0

**Release date:** 2022-04-20

This prerelease includes flux2 [v0.29.0](https://github.com/fluxcd/flux2/releases/tag/v0.29.0).

In addition, the examples have been updated to reflect the deprecation of the
`organization` field of the GitHub provider, in favour of `owner`.

## 0.12.2

**Release date:** 2022-03-30

This prerelease includes flux2 [v0.28.5](https://github.com/fluxcd/flux2/releases/tag/v0.28.5).

## 0.12.1

**Release date:** 2022-03-28

This prerelease includes flux2 [v0.28.4](https://github.com/fluxcd/flux2/releases/tag/v0.28.4).

In addition, it also makes the base URL to get the flux install manifests from
configurable in flux_install data source.

Improvements:
* Allow specifying the baseurl for flux_install data sources
  [#251](https://github.com/fluxcd/terraform-provider-flux/pull/251)

## 0.12.0

**Release date:** 2022-03-23

This prerelease includes flux2 [v0.28.2](https://github.com/fluxcd/flux2/releases/tag/v0.28.2).

Flux v0.28 comes with breaking changes, new features, and bug fixes.
Please see the [Upgrade Flux to the Source v1beta2 API](https://github.com/fluxcd/flux2/discussions/2567)
discussion for more details.

### Breaking changes

With the introduction of Source v1beta2, there is a breaking change that
requires a manual state update.

All that is required is to remove the `kubectl_manifest` resource for the
GitRepository manifest. This will cause the kubectl provider to overwrite the
existing manifest.

```shell
terraform state rm 'kubectl_manifest.sync["source.toolkit.fluxcd.io/v1beta1/gitrepository/flux-system/flux-system"]'
```

Future versions of the provider will solve this long term.

## 0.11.3

**Release date:** 2022-03-15

This prerelease includes flux2 [v0.27.4](https://github.com/fluxcd/flux2/releases/tag/v0.27.4).

## 0.11.2

**Release date:** 2022-03-01

This prerelease includes flux2 [v0.27.3](https://github.com/fluxcd/flux2/releases/tag/v0.27.3).

## 0.11.1

**Release date:** 2022-02-23

This prerelease includes flux2 [v0.27.2](https://github.com/fluxcd/flux2/releases/tag/v0.27.2).

## 0.11.0

**Release date:** 2022-02-16

This prerelease includes flux2 [v0.27.0](https://github.com/fluxcd/flux2/releases/tag/v0.27.0).

## 0.10.2

**Release date:** 2022-02-10

This prerelease includes flux2 [v0.26.3](https://github.com/fluxcd/flux2/releases/tag/v0.26.3).

## 0.10.1

**Release date:** 2022-02-07

This prerelease includes flux2 [v0.26.2](https://github.com/fluxcd/flux2/releases/tag/v0.26.2).

## 0.10.0

**Release date:** 2022-02-01

This prerelease includes flux2 [v0.26.0](https://github.com/fluxcd/flux2/releases/tag/v0.26.0).

Note that Flux v0.26 comes with breaking changes, most notable,
the minimum supported version of Kubernetes is now **v1.20.6**.
While Flux may still work on Kubernetes 1.19, we don’t recommend running EOL versions in production
as we don't run any conformance tests on Kubernetes versions that have reached end-of-life.

## 0.9.0

**Release date:** 2022-01-19

This prerelease includes flux2 [v0.25.3](https://github.com/fluxcd/flux2/releases/tag/v0.25.3).

In addition, the provider is now built with Go 1.17.

## 0.8.1

**Release date:** 2021-12-10

This prerelease includes flux2 [v0.24.1](https://github.com/fluxcd/flux2/releases/tag/v0.24.1).

## 0.8.0

**Release date:** 2021-11-24

This prerelease includes flux2 [v0.24.0](https://github.com/fluxcd/flux2/releases/tag/v0.24.0),
and allows for [enabling bootstrap customization of Flux components](https://github.com/fluxcd/terraform-provider-flux/pull/217)
through Terraform config.

## 0.7.3

**Release date:** 2021-11-12

This prerelease includes flux2 [v0.23.0](https://github.com/fluxcd/flux2/releases/tag/v0.23.0).

## 0.7.2

**Release date:** 2021-11-11

This prerelease includes flux2 [v0.22.1](https://github.com/fluxcd/flux2/releases/tag/v0.22.1).

## 0.7.1

**Release date:** 2021-11-10

This prerelease includes flux2 [v0.22.0](https://github.com/fluxcd/flux2/releases/tag/v0.22.0).

## 0.7.0

**Release date:** 2021-11-05

This prerelease includes flux2 [v0.21.1](https://github.com/fluxcd/flux2/releases/tag/v0.21.1),
and adds support for defining Tag, SemVer and Commit references in `data.flux_sync`.

## 0.6.1

**Release date:** 2021-11-01

This prerelease includes flux2 [v0.20.1](https://github.com/fluxcd/flux2/releases/tag/v0.20.1).

## 0.6.0

**Release date:** 2021-10-28

This prerelease includes flux2 [v0.20.0](https://github.com/fluxcd/flux2/releases/tag/v0.20.0).

## 0.5.1

**Release date:** 2021-10-22

This prerelease includes flux2 [v0.19.1](https://github.com/fluxcd/flux2/releases/tag/v0.19.1).

## 0.5.0

**Release date:** 2021-10-19

This prerelease includes flux2 [v0.19.0](https://github.com/fluxcd/flux2/releases/tag/v0.19.0).

## 0.4.0

**Release date:** 2021-10-15

This prerelease includes flux2 [v0.18.3](https://github.com/fluxcd/flux2/releases/tag/v0.18.3).

Flux v0.18 comes with breaking changes, new features, performance improvements and many bug fixes.

Please see the [Upgrade to Flux v0.18 and the v1beta2 API](https://github.com/fluxcd/flux2/discussions/1916) discussion for more details.

**Breaking changes**
With the introduction of Kustomization v1beta2 there is a breaking change that requires a manual state update.

All that is required is to remove the `kubectl_manifest` resource for the Kustomization manifest. This will cause the kubectl provider to overwrite the existing manifest.

```shell
terraform state rm 'kubectl_manifest.sync["kustomize.toolkit.fluxcd.io/v1beta1/kustomization/flux-system/flux-system"]'
```

Future versions of the provider will solve this long term.

## 0.3.1

**Release date:** 2021-09-13

This prerelease includes flux2 [v0.17.1](https://github.com/fluxcd/flux2/releases/tag/v0.17.1).

## 0.3.0

**Release date:** 2021-08-26

This prerelease includes flux2 [v0.17.0](https://github.com/fluxcd/flux2/releases/tag/v0.17.0).

## 0.2.2

**Release date:** 2021-08-06

This prerelease includes flux2 [v0.16.2](https://github.com/fluxcd/flux2/releases/tag/v0.16.2).

## 0.2.1

**Release date:** 2021-07-30

This prerelease includes flux2 [v0.16.1](https://github.com/fluxcd/flux2/releases/tag/v0.16.1).

## 0.2.0

**Release date:** 2021-07-01

This prerelease includes flux2 [v0.16.0](https://github.com/fluxcd/flux2/releases/tag/v0.16.0).

## 0.1.12

**Release date:** 2021-06-23

This prerelease includes flux2 [v0.15.3](https://github.com/fluxcd/flux2/releases/tag/v0.15.3).

## 0.1.11

**Release date:** 2021-06-18

This prerelease includes flux2 [v0.15.2](https://github.com/fluxcd/flux2/releases/tag/v0.15.2).

## 0.1.10

**Release date:** 2021-06-15

This prerelease includes flux2 [v0.15.0](https://github.com/fluxcd/flux2/releases/tag/v0.15.0).

## 0.1.9

**Release date:** 2021-06-03

This prerelease includes flux2 [v0.14.2](https://github.com/fluxcd/flux2/releases/tag/v0.14.2).

## 0.1.8

**Release date:** 2021-05-28

This prerelease includes flux2 [v0.14.1](https://github.com/fluxcd/flux2/releases/tag/v0.14.1).

## 0.1.7

**Release date:** 2021-05-27

This prerelease includes flux2 [v0.14.0](https://github.com/fluxcd/flux2/releases/tag/v0.14.0).

## 0.1.6

**Release date:** 2021-05-10

This prerelease includes flux2 [v0.13.4](https://github.com/fluxcd/flux2/releases/tag/v0.13.3).

## 0.1.5

**Release date:** 2021-05-10

This prerelease includes flux2 [v0.13.3](https://github.com/fluxcd/flux2/releases/tag/v0.13.3).

## 0.1.4

**Release date:** 2021-04-28

This prerelease includes flux2 [v0.13.1](https://github.com/fluxcd/flux2/releases/tag/v0.13.1).

**Breaking changes**
In this version the image automation APIs have been promoted to `v1alpha2`.
The new APIs come with breaking changes, please follow the [image automation upgrade guide](https://github.com/fluxcd/flux2/discussions/1333).

## 0.1.3

**Release date:** 2021-04-08

Remove `image-reflector-controller` and `image-automation-controller` as default values
for `components_extra` to reflect the behavior of the CLI.

## 0.1.2

**Release date:** 2021-04-08

This prerelease includes flux2 [v0.12.0](https://github.com/fluxcd/flux2/releases/tag/v0.12.0).

## 0.1.1

**Release date:** 2021-03-26

This prerelease includes flux2 [v0.11.0](https://github.com/fluxcd/flux2/releases/tag/v0.11.0).

## 0.1.0

**Release date:** 2021-03-18

This prerelease includes flux2 [v0.10.0](https://github.com/fluxcd/flux2/releases/tag/v0.10.0).

## 0.0.14

**Release date:** 2021-03-17

This prerelease adds two new properties to new properties, one
for each datasource. A notable change is that the name property in
data sync no longer set the sync secret name. Instead this should
be done with the secret property.

Improvements:
* Add components extra to data sync data source
  [#115](https://github.com/fluxcd/terraform-provider-flux/pull/115)
* Make secret ref configurable
  [#113](https://github.com/fluxcd/terraform-provider-flux/pull/113)
* Add an example for GKE and Github
  [#112](https://github.com/fluxcd/terraform-provider-flux/pull/112)

## 0.0.13

**Release date:** 2021-03-09

This prerelease includes flux2 [v0.9.1](https://github.com/fluxcd/flux2/releases/tag/v0.9.1)

## 0.0.12

**Release date:** 2021-02-12

A new `toleration_keys` parameter has been added to the install
data source to enable installtion in cluster with node taints.

Improvements:
* Add tolerations parameter to install datasource
  [#96](https://github.com/fluxcd/terraform-provider-flux/pull/96)
* Fix resource name
  [#90](https://github.com/fluxcd/terraform-provider-flux/pull/90)

## 0.0.11

**Release date:** 2021-01-29

The example guides have changed to improve the upgrading experience.
The resource keys will change from being based on the content applied
to the identifier of the resource being applied. This change will
make moving from one version of Flux to another more stable.

Improvements:
* Refactor provider to reliably track changes in manifestgen output
  [#85](https://github.com/fluxcd/terraform-provider-flux/pull/85)
* Add verification and check of image versions
  [#81](https://github.com/fluxcd/terraform-provider-flux/pull/81)

## 0.0.10

**Release date:** 2021-01-15

The `arch` parameters has been removed from the `flux_install` resource
as multi-arch images are now published under the same tag.

Improvements:
* Update flux to v0.6.0
  [#77](https://github.com/fluxcd/terraform-provider-flux/pull/77)
* Add git implementation to sync datasource
  [#76](https://github.com/fluxcd/terraform-provider-flux/pull/76)
