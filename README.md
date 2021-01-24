# terraform-provider-flux

This is the Terraform provider for Flux v2.
The provider allows you to install Flux on Kubernetes
and configure it to reconcile the cluster state from a Git repository.

## Example Usage

The provider consists of two data sources `flux_install` and `flux_sync`,
the data sources are corresponding to [fluxv2 manifests](https://pkg.go.dev/github.com/fluxcd/flux2@v0.2.1/pkg/manifestgen).

The `flux_install` data source generates a multi-doc YAML with Kubernetes manifests that can be used to install or upgrade Flux:

```hcl
# Generate manifests
data "flux_install" "main" {
  target_path    = "production"
  network_policy = false
  version        = "latest"
}

resource "kubernetes_namespace" "flux_system" {
  metadata {
    name = "flux-system"
  }

  lifecycle {
    ignore_changes = [
      metadata[0].labels,
    ]
  }
}

# Split multi-doc YAML with
# https://registry.terraform.io/providers/gavinbunney/kubectl/latest
data "kubectl_file_documents" "apply" {
  content = data.flux_install.main.content
}

# Apply manifests on the cluster
resource "kubectl_manifest" "apply" {
  for_each  = { for v in data.kubectl_file_documents.apply.documents : sha1(v) => v }
  depends_on = [kubernetes_namespace.flux_system]

  yaml_body = each.value
}
```

The `flux_sync` data source generates a multi-doc YAML containing the `GitRepository` and `Kustomization`
manifests that configure Flux to sync the cluster with the specified repository:

```hcl
# Generate manifests
data "flux_sync" "main" {
  target_path = "production"
  url         = "https://github.com/${var.github_owner}/${var.repository_name}"
}

# Split multi-doc YAML with
# https://registry.terraform.io/providers/gavinbunney/kubectl/latest
data "kubectl_file_documents" "sync" {
  content = data.flux_sync.main.content
}

# Apply manifests on the cluster
resource "kubectl_manifest" "sync" {
  for_each  = { for v in data.kubectl_file_documents.sync.documents : sha1(v) => v }
  depends_on = [kubectl_manifest.apply, kubernetes_namespace.flux_system]

  yaml_body = each.value
}

# Generate a Kubernetes secret with the Git credentials
resource "kubernetes_secret" "main" {
  depends_on = [kubectl_manifest.apply]

  metadata {
    name      = data.flux_sync.main.name
    namespace = data.flux_sync.main.namespace
  }

  data = {
    username = "git"
    password = var.flux_token
  }
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
