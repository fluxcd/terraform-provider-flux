# Gitlab via SSH

The example demonstrates how to bootstrap a KinD cluster with Flux using a Gitlab project via SSH.

Note: The Gitlab project is created and initialised with a README.md file ready for Flux to use.

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.7.0 |
| <a name="requirement_flux"></a> [flux](#requirement\_flux) | >= 1.2 |
| <a name="requirement_gitlab"></a> [gitlab](#requirement\_gitlab) | >= 16.10 |
| <a name="requirement_kind"></a> [kind](#requirement\_kind) | >= 0.4 |
| <a name="requirement_tls"></a> [tls](#requirement\_tls) | >= 4.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_flux"></a> [flux](#provider\_flux) | >= 1.2 |
| <a name="provider_gitlab"></a> [gitlab](#provider\_gitlab) | >= 16.10 |
| <a name="provider_kind"></a> [kind](#provider\_kind) | >= 0.4 |
| <a name="provider_tls"></a> [tls](#provider\_tls) | >= 4.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [flux_bootstrap_git.this](https://registry.terraform.io/providers/fluxcd/flux/latest/docs/resources/bootstrap_git) | resource |
| [gitlab_deploy_key.this](https://registry.terraform.io/providers/gitlabhq/gitlab/latest/docs/resources/deploy_key) | resource |
| [gitlab_project.this](https://registry.terraform.io/providers/gitlabhq/gitlab/latest/docs/resources/project) | resource |
| [kind_cluster.this](https://registry.terraform.io/providers/tehcyx/kind/latest/docs/resources/cluster) | resource |
| [tls_private_key.flux](https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_gitlab_project"></a> [gitlab\_project](#input\_gitlab\_project) | The GitLab project to use for creating the GitLab project. | `string` | `""` | no |
| <a name="input_gitlab_token"></a> [gitlab\_token](#input\_gitlab\_token) | The GitLab token to use for authenticating against the GitLab API. | `string` | `""` | no |

## Outputs

No outputs.
<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
