resource "github_repository_file" "kustomize" {
  repository = github_repository.main.name
  file       = data.flux_sync.main.kustomize_path
  content    = file("${path.module}/templates/kustomization-override.yaml")
  branch     = var.branch
}
