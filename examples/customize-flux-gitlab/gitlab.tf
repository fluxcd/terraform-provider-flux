resource "gitlab_project" "main" {
  name                   = var.repository_name
  visibility_level       = var.repository_visibility
  initialize_with_readme = true
  default_branch         = var.branch
}

resource "gitlab_deploy_key" "main" {
  title   = "staging-cluster"
  project = gitlab_project.main.id
  key     = tls_private_key.main.public_key_openssh

  depends_on = [gitlab_project.main]
}

resource "gitlab_repository_file" "install" {
  project        = gitlab_project.main.id
  branch         = gitlab_project.main.default_branch
  file_path      = data.flux_install.main.path
  content        = base64encode(data.flux_install.main.content)
  commit_message = "Add ${data.flux_install.main.path}"

  depends_on = [gitlab_project.main]
}

resource "gitlab_repository_file" "sync" {
  project        = gitlab_project.main.id
  branch         = gitlab_project.main.default_branch
  file_path      = data.flux_sync.main.path
  content        = base64encode(data.flux_sync.main.content)
  commit_message = "Add ${data.flux_sync.main.path}"

  depends_on = [gitlab_repository_file.install]
}

resource "gitlab_repository_file" "kustomize" {
  project        = gitlab_project.main.id
  branch         = gitlab_project.main.default_branch
  file_path      = data.flux_sync.main.kustomize_path
  content        = base64encode(data.flux_sync.main.kustomize_content)
  commit_message = "Add ${data.flux_sync.main.kustomize_path}"

  depends_on = [gitlab_repository_file.sync]
}

resource "gitlab_repository_file" "patches" {
  for_each = data.flux_sync.main.patch_file_paths

  project        = gitlab_project.main.id
  file_path      = each.value
  content        = yamlencode(local.patches[each.key])
  branch         = var.branch
  commit_message = "Add ${each.key}"
}

