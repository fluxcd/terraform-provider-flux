
terraform {
  required_version = ">= 0.13"

  required_providers {
    flux = {
      source = "fluxcd/flux"
    }
    gitfile = {
      source  = "igal-s/gitfile"
      version = "1.0.0"
    }
  }
}

provider "flux" {}
provider "gitfile" {}

# Generate the flux manifest
data "flux_install" "main" {
  target_path = "cluster-init"
}

data "flux_sync" "main" {
  target_path = "cluster-init"
  url         = var.repository_url
}

# Git: Create the files and commit them to the repository
resource "gitfile_checkout" "main" {
  repo   = var.repository_url
  branch = var.branch
  path   = "${path.module}/git_checkouts/cluster-init"
}

resource "gitfile_file" "install" {
  checkout_dir = gitfile_checkout.example.path
  path         = data.flux_install.main.path
  contents     = data.flux_install.main.content
}

resource "gitfile_file" "sync" {
  checkout_dir = gitfile_checkout.example.path
  path         = data.flux_sync.main.path
  contents     = data.flux_sync.main.content
}

resource "gitfile_commit" "flux_install" {
  checkout_dir   = gitfile_checkout.example.path
  commit_message = "Added install from flux provider"
  handle         = gitfile_file.install.id
}

resource "gitfile_commit" "flux_sync" {
  checkout_dir   = gitfile_checkout.example.path
  commit_message = "Added sync from flux provider"
  handle         = gitfile_file.sync.id
}
