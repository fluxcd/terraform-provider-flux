terraform {
  required_version = ">= 0.13"

  required_providers {
    github = {
      source = "integrations/github"
      version = "4.5.2"
    }
  }
}

resource "github_repository_file" "kustomize" {
  repository = github_repository.main.name
  file       = data.flux_sync.main.kustomize_path
  content    = file("${path.module}/kustomization-override.yaml")
  branch     = var.branch
}

resource "github_repository_file" "psp_patch" {
  repository = github_repository.main.name
  file       = "${dirname(data.flux_sync.main.kustomize_path)}/psp-patch.yaml"
  content    = file("${path.module}/psp-patch.yaml")
  branch     = var.branch
}
