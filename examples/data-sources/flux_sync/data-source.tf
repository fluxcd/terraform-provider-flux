variable "target_path" {
  type = string
}

variable "clone_url" {
  type = string
}

data "flux_sync" "main" {
  target_path = var.target_path
  url         = var.clone_url
}
