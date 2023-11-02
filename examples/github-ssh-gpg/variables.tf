variable "flux_gpg_key_id" {
  type = string
  description = "The ID of the GPG key to use for signing commits when bootstraping FluxCD."
}

variable "flux_gpg_key_ring" {
  type = string
  description = "The path to the exported GPG key ring."
}

variable "flux_gpg_passphrase" {
  sensitive = true
  type = string
  description = "The passphrase of the GPG key."
  default = ""
}

variable "github_token" {
  sensitive = true
  type      = string
  description = "The GitHub token to use for authenticating with the GitHub API."
}

variable "github_org" {
  type = string
  description = "The name of the GitHub organization/username for the repository."
}

variable "github_repository" {
  type = string
  description = "The name of the GitHub repository to create the FluxCD manifests in."
}
