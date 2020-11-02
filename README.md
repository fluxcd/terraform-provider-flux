# terraform-provider-flux

This is a Terraform provider for Flux v2, it enables bootstrap a Kubernetes custer with Flux v2 using terraform.

## Example Usage

The provider is consists of two data sources `flux_install` and `flux_sync` the data sources are corresponding to [flux manifests](https://pkg.go.dev/github.com/fluxcd/flux2@v0.2.1/pkg/manifestgen)

The data sources are returing `YAML` manifest so a second provider is needed to apply the manifest into the Kubernetes cluster. See example folder.

The `flux_install` generates manifests to install the `flux` components.

```hcl
# Flux
data "flux_install" "main" {
  target_path = "staging-cluster"
}
```

`flux_sync` the initial source manifest.

```hcl
data "flux_sync" "main" {
  target_path = "staging-cluster"
  url         = "ssh://git@github.com/${var.github_owner}/${var.repository_name}.git"
}
```

## Community

The Flux project is always looking for new contributors and there are a multitude of ways to get involved.
Depending on what you want to do, some of the following bits might be your first steps:

- Join our upcoming dev meetings ([meeting access and agenda](https://docs.google.com/document/d/1l_M0om0qUEN_NNiGgpqJ2tvsF2iioHkaARDeh6b70B0/view))
- Talk to us in the #flux channel on [CNCF Slack](https://slack.cncf.io/)
- Join the [planning discussions](https://github.com/fluxcd/flux2/discussions)
- And if you are completely new to Flux and the GitOps Toolkit, take a look at our [Get Started guide](https://toolkit.fluxcd.io/get-started/) and give us feedback
- To be part of the conversation about Flux's development, [join the flux-dev mailing list](https://lists.cncf.io/g/cncf-flux-dev).
- Check out [how to contribute](CONTRIBUTING.md) to the project
