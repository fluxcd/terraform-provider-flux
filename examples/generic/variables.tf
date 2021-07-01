variable "repository_url" {
  type        = string
  description = "The url for the git repo. Example: ssh://git@codeberg.org/<user>/fluxv2.git "
}

variable "branch" {
  type    = string
  default = "main"
}
