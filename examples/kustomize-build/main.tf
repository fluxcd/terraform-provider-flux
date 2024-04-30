locals {
  flux_manifests_path = abspath(var.flux_root_dir)
}

data "kustomization_build" "flux" {
  path = local.flux_manifests_path
}

resource "kubernetes_manifest" "flux" {
  for_each = data.kustomization_build.flux.manifests
  manifest = each.value
}
