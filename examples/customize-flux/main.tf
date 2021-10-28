terraform {
  required_version = ">= 0.13"

  required_providers {
    github = {
      source = "integrations/github"
      version = "4.5.2"
    }
  }
}

variable "target_path" {
  type = string
}

variable "clone_url" {
  type = string
}

data "google_service_account" "flux_sops" {
  account_id = "flux-sops"
}

locals {
  # 'Small patches that do one thing are recommended'
  #   - https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/#customizing
  patches = {
    psp  = file("./psp-patch.yaml")
    sops = <<EOT
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kustomize-controller
  namespace: flux-system
  annotations:
    iam.gke.io/gcp-service-account: ${data.google_service_account.flux_sops.email}
EOT
  }
}

data "flux_sync" "main" {
  target_path = var.target_path
  url         = var.clone_url
  patch_names = keys(local.patches)
}

# Create kustomize.yaml
resource "github_repository_file" "kustomize" {
  repository          = local.repository_name
  file                = data.flux_sync.flux.kustomize_path
  content             = data.flux_sync.flux.kustomize_content
  branch              = data.github_branch.main.branch
  overwrite_on_create = true
}

resource "github_repository_file" "patches" {
  #  `patch_file_paths` is a map keyed by the keys of `flux_sync.main`
  #  whose values are the paths where the patch files should be installed.
  for_each   = data.flux_sync.main.patch_file_paths

  repository = github_repository.main.name
  file       = each.value
  content    = local.patches[each.key] # Get content of our patch files
  branch     = var.branch
}
