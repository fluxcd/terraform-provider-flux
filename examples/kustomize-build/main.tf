locals {
  flux_manifests_path = format("%s/flux-system", abspath(var.flux_root_dir))
}

## NB: this requires jq and kustomize to be available
## there is https://registry.terraform.io/providers/kbst/kustomization/latest/docs/data-sources/build
## which works, but it requires kubeconfig to be passed in, while API access is not
## required for kustomize build to work, perhaps this provider can be patched in the
## future to avoid this hack
data "external" "kustomize_build" {
  program = ["sh", "-e", "-u", "-c", format("output=\"$(mktemp)\" && kustomize build %s --output \"$${output}\" && jq -n --rawfile output \"$${output}\" '{ data: $output|@base64 }' && rm -f \"$${output}\"", local.flux_manifests_path)]
}

## hashicorp/kubernetes provider has kubernetes_manifest resource that leverages
## server-side apply, as well as provider::kubernetes::manifest_decode_multi()
## function, however it requires CRDs to be registered before any CRs can be
## planned and namespaces to be defined separately as well (which conflict with
## having the namespaces in a manifest); however what is really hard to overcome
## is that hashicorp/kubernetes has an ultimate desire to track every single
## field of every single resource using its own type system, which can result
## in it being greatly confused when something like CPU units get converted (see
## https://github.com/hashicorp/terraform-provider-kubernetes/issues/1466, which
## is presently archived, yet still unresolved)
data "kubectl_file_documents" "flux" {
  content = base64decode(data.external.kustomize_build.result.data)
}

resource "kubectl_manifest" "flux" {
  for_each  = data.kubectl_file_documents.flux.manifests
  yaml_body = each.value

  server_side_apply = true
}
