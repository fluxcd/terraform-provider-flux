variable "target_path" {
  type = string
}

data "flux_install" "main" {
  target_path = var.target_path
}
