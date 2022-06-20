locals {
  deployments = ["helm-controller", "kustomize-controller", "notification-controller", "source-controller"]
  patches = {
    for resource_name in local.deployments : resource_name => yamldecode(templatefile("./psp-patch.tftpl", {
      deployment_name = "${resource_name}"
    }))
  }
}

data "flux_install" "main" {
  target_path = var.target_path
}

data "flux_sync" "main" {
  target_path = var.target_path
  url         = "ssh://git@gitlab.com/${var.gitlab_owner}/${var.repository_name}.git"
  branch      = var.branch
  patch_names = keys(local.patches)
}
