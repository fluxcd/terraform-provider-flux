# Forgejo via SSH

The example demonstrates how to bootstrap a KinD cluster with Flux using a Forgejo repository via SSH.

Note: The Forgejo repository is created and auto initialised ready for Flux to use.

<!-- BEGINNING OF PRE-COMMIT-TERRAFORM DOCS HOOK -->

## Requirements

| Name                                                                     | Version  |
| ------------------------------------------------------------------------ | -------- |
| <a name="requirement_terraform"></a> [terraform](#requirement_terraform) | >= 1.7.0 |
| <a name="requirement_flux"></a> [flux](#requirement_flux)                | >= 1.5   |
| <a name="requirement_forgejo"></a> [forgejo](#requirement_forgejo)       | >= 0.2   |
| <a name="requirement_kind"></a> [kind](#requirement_kind)                | >= 0.8   |
| <a name="requirement_tls"></a> [tls](#requirement_tls)                   | >= 4.0   |

## Providers

| Name                                                         | Version |
| ------------------------------------------------------------ | ------- |
| <a name="provider_flux"></a> [flux](#provider_flux)          | >= 1.5  |
| <a name="provider_forgejo"></a> [forgejo](#provider_forgejo) | >= 0.2  |
| <a name="provider_kind"></a> [kind](#provider_kind)          | >= 0.8  |
| <a name="provider_tls"></a> [tls](#provider_tls)             | >= 4.0  |

## Modules

No modules.

## Resources

| Name                                                                                                                | Type     |
| ------------------------------------------------------------------------------------------------------------------- | -------- |
| [flux_bootstrap_git.this](https://registry.terraform.io/providers/fluxcd/flux/latest/docs/resources/bootstrap_git)  | resource |
| [forgejo_deploy_key.this](https://registry.terraform.io/providers/svalabs/forgejo/latest/docs/resources/deploy_key) | resource |
| [forgejo_repository.this](https://registry.terraform.io/providers/svalabs/forgejo/latest/docs/resources/repository) | resource |
| [kind_cluster.this](https://registry.terraform.io/providers/tehcyx/kind/latest/docs/resources/cluster)              | resource |
| [tls_private_key.ed25519](https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key)  | resource |

## Inputs

| Name                                                                                    | Description          | Type     | Default       | Required |
| --------------------------------------------------------------------------------------- | -------------------- | -------- | ------------- | :------: |
| <a name="input_forgejo_host"></a> [forgejo_host](#input_forgejo_host)                   | Forgejo hostname     | `string` | `"localhost"` |    no    |
| <a name="input_forgejo_port"></a> [forgejo_port](#input_forgejo_port)                   | Forgejo port         | `number` | 3000          |    no    |
| <a name="input_forgejo_token"></a> [forgejo_token](#input_forgejo_token)                | Forgejo API token    | `string` | `""`          |    no    |
| <a name="input_forgejo_org"></a> [forgejo_org](#input_forgejo_org)                      | Forgejo organization | `string` | `""`          |    no    |
| <a name="input_forgejo_repository"></a> [forgejo_repository](#input_forgejo_repository) | Forgejo repository   | `string` | `""`          |    no    |

## Outputs

No outputs.

<!-- END OF PRE-COMMIT-TERRAFORM DOCS HOOK -->
