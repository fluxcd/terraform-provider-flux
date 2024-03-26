variable "flux_version" {
  description = "Flux version"
  type        = string
  default     = "2.2.3"
}

variable "gpg_key_id" {
  description = "The ID of the GPG key to use for signing commits when bootstraping FluxCD."
  type        = string
  default     = ""
}

variable "gpg_key_ring" {
  description = "The path to the exported GPG key ring."
  type        = string
  default     = ""
}

variable "gpg_passphrase" {
  description = "The passphrase of the GPG key."
  sensitive   = true
  type        = string
  default     = ""
}

variable "github_token" {
  description = "GitHub token"
  sensitive   = true
  type        = string
  default     = ""
}

variable "github_org" {
  description = "GitHub organization"
  type        = string
  default     = ""
}

variable "github_repository" {
  description = "GitHub repository"
  type        = string
  default     = ""
}
