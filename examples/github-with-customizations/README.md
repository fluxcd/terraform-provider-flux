# GitHub via SSH (with flux customizations)

The example demonstrates how to bootstrap a KinD cluster with flux using a GitHub repository via SSH with customizations provided.

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | = 1.7.0 |
| <a name="requirement_flux"></a> [flux](#requirement\_flux) | 1.2.3 |
| <a name="requirement_github"></a> [github](#requirement\_github) | 6.1.0 |
| <a name="requirement_kind"></a> [kind](#requirement\_kind) | 0.4.0 |
| <a name="requirement_tls"></a> [tls](#requirement\_tls) | 4.0.5 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_flux"></a> [flux](#provider\_flux) | 1.2.3 |
| <a name="provider_github"></a> [github](#provider\_github) | 6.1.0 |
| <a name="provider_kind"></a> [kind](#provider\_kind) | 0.4.0 |
| <a name="provider_tls"></a> [tls](#provider\_tls) | 4.0.5 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [flux_bootstrap_git.this](https://registry.terraform.io/providers/fluxcd/flux/1.2.3/docs/resources/bootstrap_git) | resource |
| [github_repository_deploy_key.this](https://registry.terraform.io/providers/integrations/github/6.1.0/docs/resources/repository_deploy_key) | resource |
| [kind_cluster.this](https://registry.terraform.io/providers/tehcyx/kind/0.4.0/docs/resources/cluster) | resource |
| [tls_private_key.flux](https://registry.terraform.io/providers/hashicorp/tls/4.0.5/docs/resources/private_key) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_github_org"></a> [github\_org](#input\_github\_org) | GitHub organization | `string` | `""` | no |
| <a name="input_github_repository"></a> [github\_repository](#input\_github\_repository) | GitHub repository | `string` | `""` | no |
| <a name="input_github_token"></a> [github\_token](#input\_github\_token) | GitHub token | `string` | `""` | no |

## Outputs

No outputs.
<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
