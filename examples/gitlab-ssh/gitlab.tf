provider "gitlab" {
  token = var.gitlab_token
}

resource "tls_private_key" "flux" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P256"
}

data "gitlab_project" "this" {
  path_with_namespace = "${var.gitlab_group}/${var.gitlab_project}"
}

resource "gitlab_deploy_key" "this" {
  project  = data.gitlab_project.this.id
  title    = "Flux"
  key      = tls_private_key.flux.public_key_openssh
  can_push = true
}
