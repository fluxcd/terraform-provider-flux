variable "gitlab_owner" {
  description = "gitlab owner"
  type = string
}

variable "gitlab_token" {
  description = "gitlab token"
  type      = string
  sensitive = true
}

variable "repository_name" {
  description = "gitlab repository name"
  type        = string
  default     = "test-provider"
}

variable "repository_visibility" {
  description = "how visible is the gitlab repo"
  type        = string
  default     = "private"
}

variable "branch" {
  type    = string
  default = "main"
}

variable "target_path" {
  type    = string
  default = "staging-cluster"
}

