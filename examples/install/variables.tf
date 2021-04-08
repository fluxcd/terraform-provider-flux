variable "target_path" {
  type        = string
  default     = "staging-cluster"
  description = "flux install target path"
}

variable "components_extra" {
  type        = list(string)
  default     =  []
  description = "extra flux components"
}
