provider "flux" {}

provider "kubectl" {}

provider "kubernetes" {
  config_path = "~/.kube/config"
}

provider "gitlab" {
  token = var.gitlab_token
}

