# Copyright (c) The Flux authors
# SPDX-License-Identifier: Apache-2.0

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
