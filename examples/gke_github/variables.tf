variable "project_id" {
  type = string
}

variable "github_token" {
  type = string
}

variable "repository_name" {
  type = string
}

variable "organization" {
  type = string
}

variable "branch" {
  type    = string
  default = "main"
}

variable "target_path" {
  type        = string
  description = "Relative path to the Git repository root where the sync manifests are committed."
}

variable "flux_namespace" {
  type    = string
  default = "flux-system"
}

variable "cluster_name" {
  type = string
}

variable "cluster_region" {
  type = string
}

variable "use_private_endpoint" {
  type        = bool
  description = "Connect on the private GKE cluster endpoint"
  default     = false
}
