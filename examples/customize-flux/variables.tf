variable "target_path" {
  type        = string
  default     = "staging-cluster"
  description = "flux sync target path"
}


variable "clone_url" {
  type = string
  description = "Git repository clone url"
}

