# Copyright (c) The Flux authors
# SPDX-License-Identifier: Apache-2.0

variable "gpg_key_id" {
  description = "The ID of the GPG key to use for signing commits when bootstraping FluxCD."
  type        = string
  default     = ""
}

variable "gpg_key_ring" {
  description = "The path to the exported GPG key ring."
  type        = string
  default     = ""
}

variable "gpg_passphrase" {
  description = "The passphrase of the GPG key."
  sensitive   = true
  type        = string
  default     = ""
}

variable "gitlab_token" {
  description = "The GitLab token to use for authenticating against the GitLab API."
  sensitive   = true
  type        = string
  default     = ""
}

variable "gitlab_group" {
  description = "The GitLab group to use for creating the GitLab project."
  type        = string
  default     = ""
}

variable "gitlab_project" {
  description = "The GitLab project to use for creating the GitLab project."
  type        = string
  default     = ""
}
