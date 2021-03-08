variable "repository_name" {
  description = "Name of GitHub repository"
  type        = string
}

variable "branch" {
  description = "Name of git branch"
  type        = string
}

variable "environment" {
  description = "Environment being deployed"
  type        = string
}
