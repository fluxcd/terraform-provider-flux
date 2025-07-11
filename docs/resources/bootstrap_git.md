---
page_title: "flux_bootstrap_git Resource - terraform-provider-flux"
subcategory: ""
description: |-
  Commits Flux components to a Git repository and configures a Kubernetes cluster to synchronize with the same Git repository.
---

# flux_bootstrap_git (Resource)

Commits Flux components to a Git repository and configures a Kubernetes cluster to synchronize with the same Git repository.

## Example Usage

The following examples are available to help you use the provider:

- [Bootstrapping a cluster using a GitHub repository and a personal access token (PAT)](https://github.com/fluxcd/terraform-provider-flux/tree/main/examples/github-via-pat)
- [Bootstrapping a cluster using a GitHub repository via SSH](https://github.com/fluxcd/terraform-provider-flux/tree/main/examples/github-via-ssh)
- [Bootstrapping a cluster using a GitHub repository via SSH and GPG](https://github.com/fluxcd/terraform-provider-flux/tree/main/examples/github-via-ssh-with-gpg)
- [Bootstrapping a cluster using a GitHub repository via SSH with flux customizations](https://github.com/fluxcd/terraform-provider-flux/tree/main/examples/github-with-customizations)
- [Bootstrapping a cluster using a GitHub repository via SSH and GPG with inline flux customizations](https://github.com/fluxcd/terraform-provider-flux/tree/main/examples/github-with-inline-customizations)
- [Bootstrapping a cluster using a Gitlab repository via SSH](https://github.com/fluxcd/terraform-provider-flux/tree/main/examples/gitlab-via-ssh)
- [Bootstrapping a cluster using a Gitlab repository via SSH and GPG](https://github.com/fluxcd/terraform-provider-flux/tree/main/examples/gitlab-via-ssh-with-gpg)
- [Bootstrapping a cluster using a Forgejo repository via SSH](https://github.com/fluxcd/terraform-provider-flux/tree/main/examples/forgejo-via-ssh)
- [Bootstrapping a cluster using a Helm Release](https://github.com/fluxcd/terraform-provider-flux/tree/main/examples/helm-install)

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `cluster_domain` (String) The internal cluster domain. Defaults to `cluster.local`
- `components` (Set of String) Toolkit components to include in the install manifests. Defaults to `[source-controller kustomize-controller helm-controller notification-controller]`
- `components_extra` (Set of String) List of extra components to include in the install manifests.
- `delete_git_manifests` (Boolean) Delete manifests from git repository. Defaults to `true`.
- `disable_secret_creation` (Boolean) Use the existing secret for flux controller and don't create one from bootstrap
- `embedded_manifests` (Boolean) When enabled, the Flux manifests will be extracted from the provider binary instead of being downloaded from GitHub.com. Defaults to `false`.
- `image_pull_secret` (String) Kubernetes secret name used for pulling the toolkit images from a private registry.
- `interval` (String) Interval at which to reconcile from bootstrap repository. Defaults to `1m0s`.
- `keep_namespace` (Boolean) Keep the namespace after uninstalling Flux components. Defaults to `false`.
- `kustomization_override` (String) Kustomization to override configuration set by default.
- `log_level` (String) Log level for toolkit components. Defaults to `info`.
- `manifests_path` (String, Deprecated) The install manifests are built from a GitHub release or kustomize overlay if using a local path. Defaults to `https://github.com/fluxcd/flux2/releases`.
- `namespace` (String) The namespace scope for install manifests. Defaults to `flux-system`. It will be created if it does not exist.
- `network_policy` (Boolean) Deny ingress access to the toolkit controllers from other namespaces using network policies. Defaults to `true`.
- `path` (String) Path relative to the repository root, when specified the cluster sync will be scoped to this path (immutable).
- `recurse_submodules` (Boolean) Configures the GitRepository source to initialize and include Git submodules in the artifact it produces.
- `registry` (String) Container registry where the toolkit images are published. Defaults to `ghcr.io/fluxcd`.
- `registry_credentials` (String) Container registry credentials in the format 'user:password'
- `secret_name` (String) Name of the secret the sync credentials can be found in or stored to. Defaults to `flux-system`.
- `timeouts` (Attributes) (see [below for nested schema](#nestedatt--timeouts))
- `toleration_keys` (Set of String) List of toleration keys used to schedule the components pods onto nodes with matching taints.
- `version` (String) Flux version. Defaults to `v2.6.4`. Has no effect when `embedded_manifests` is enabled.
- `watch_all_namespaces` (Boolean) If true watch for custom resources in all namespaces. Defaults to `true`.

### Read-Only

- `id` (String) The ID of this resource.
- `repository_files` (Map of String) Git repository files created and managed by the provider.

<a id="nestedatt--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).
- `delete` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Setting a timeout for a Delete operation is only applicable if changes are saved into state before the destroy operation occurs.
- `read` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Read operations occur during any refresh or planning operation when refresh is enabled.
- `update` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).

## Import

Existing Flux installations can be imported by passing the namespace where Flux is installed.

```shell
terraform import flux_bootstrap_git.this flux-system
```
