# GitHub via SSH

The example demonstrates how to bootstrap a KinD cluster with Flux using a GitHub repository via a [personal access token (PAT)](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens).

We recommend creating a fine-gained PAT and dedicated Flux user, for more information see [here](https://fluxcd.io/flux/installation/bootstrap/github/#github-organization)

Note: The GitHub repository is created and auto initialised ready for Flux to use.

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.7.0 |
| <a name="requirement_flux"></a> [flux](#requirement\_flux) | >= 1.2 |
| <a name="requirement_github"></a> [github](#requirement\_github) | >= 6.1 |
| <a name="requirement_kind"></a> [kind](#requirement\_kind) | >= 0.4 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_flux"></a> [flux](#provider\_flux) | >= 1.2 |
| <a name="provider_github"></a> [github](#provider\_github) | >= 6.1 |
| <a name="provider_kind"></a> [kind](#provider\_kind) | >= 0.4 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [flux_bootstrap_git.this](https://registry.terraform.io/providers/fluxcd/flux/latest/docs/resources/bootstrap_git) | resource |
| [github_repository.this](https://registry.terraform.io/providers/integrations/github/latest/docs/resources/repository) | resource |
| [kind_cluster.this](https://registry.terraform.io/providers/tehcyx/kind/latest/docs/resources/cluster) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_github_org"></a> [github\_org](#input\_github\_org) | GitHub organization | `string` | `""` | no |
| <a name="input_github_repository"></a> [github\_repository](#input\_github\_repository) | GitHub repository | `string` | `""` | no |
| <a name="input_github_token"></a> [github\_token](#input\_github\_token) | GitHub token | `string` | `""` | no |

## Outputs

No outputs.
<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
