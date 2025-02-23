variable "forgejo_host" {
  description = "Forgejo hostname"
  type        = string
  default     = "localhost"
}

variable "forgejo_port" {
  description = "Forgejo port"
  type        = number
  default     = 3000
}

variable "forgejo_token" {
  description = "Forgejo API token"
  sensitive   = true
  type        = string
  default     = ""
}

variable "forgejo_org" {
  description = "Forgejo organization"
  type        = string
  default     = ""
}

variable "forgejo_repository" {
  description = "Forgejo repository"
  type        = string
  default     = ""
}
