variable "gitlab_token" {
  sensitive = true
  type      = string
}

variable "gitlab_group" {
  type = string
}

variable "gitlab_project" {
  type = string
}
