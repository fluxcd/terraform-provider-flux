# Flux provider for Terraform

[![tests](https://github.com/fluxcd/terraform-provider-flux/workflows/tests/badge.svg)](https://github.com/fluxcd/terraform-provider-flux/actions)
[![report](https://goreportcard.com/badge/github.com/fluxcd/terraform-provider-flux)](https://goreportcard.com/report/github.com/fluxcd/terraform-provider-flux)
[![license](https://img.shields.io/github/license/fluxcd/terraform-provider-flux.svg)](https://github.com/fluxcd/terraform-provider-flux/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/fluxcd/terraform-provider-flux/all.svg)](https://github.com/fluxcd/terraform-provider-flux/releases)

## Overview
The Flux provider for Terraform is a plugin that enables bootstrapping of your Kubernetes cluster using [Flux v2](https://github.com/fluxcd/flux2/tree/main).

Please note: We take security and our users' trust very seriously. If you believe you have found a security issue in the Terraform Flux Provider, please follow the policy located [here](https://github.com/fluxcd/terraform-provider-flux/security/policy).

## Documentation
All documentation is available on the [Terraform provider website](https://registry.terraform.io/providers/fluxcd/flux/latest/docs).

## Guides

The following guides are available to help you use the provider:

- [Bootstrapping a cluster using a GitHub repository using a personal access token (PAT)](examples/github-via-pat)
- [Bootstrapping a cluster using a GitHub repository via SSH](examples/github-via-ssh)
- [Bootstrapping a cluster using a GitHub repository via SSH and GPG](examples/github-via-ssh-with-gpg)
- [Bootstrapping a cluster using a GitHub repository self-managing the SSH keypair secret)](examples/github-self-managed-ssh-keypair)
- [Bootstrapping a cluster using a GitHub repository via SSH with flux customizations](examples/github-with-customizations)
- [Bootstrapping a cluster using a GitHub repository via SSH and GPG with inline flux customizations](examples/github-with-inline-customizations)
- [Bootstrapping a cluster using a GitLab repository via SSH](examples/gitlab-via-ssh)
- [Bootstrapping a cluster using a GitLab repository via SSH and GPG](examples/gitlab-via-ssh-with-gpg)
- [Bootstrapping a cluster using a Forgejo repository via SSH](examples/forgejo-via-ssh)
- [Bootstrapping a cluster using a Helm Release and not the flux_bootstrap_git resource](examples/helm-install) **

** This is the recommended approach if you do not want to perform initial flux bootstrapping.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) 1.5.x or newer
- [OpenTofu](https://opentofu.org/) v1.7.x or newer
- [Go](https://golang.org/doc/install) 1.22 (to build the provider plugin)

## Contributing to the provider

The Flux Provider for Terraform is the work of many contributors. We appreciate your help!

To contribute, please read the [contribution guidelines](CONTRIBUTING.md). You may also [report an issue](https://github.com/fluxcd/terraform-provider-flux/issues/new/choose).

## Community

Need help or want to contribute? Please see the links below. The Flux project is always looking for
new contributors and there are a multitude of ways to get involved.

- Getting Started?
    - Look at our [Get Started guide](https://fluxcd.io/flux/get-started/) and give us feedback
- Need help?
    - First: Ask questions on our [GH Discussions page](https://github.com/fluxcd/flux2/discussions).
    - Second: Talk to us in the #flux channel on [CNCF Slack](https://slack.cncf.io/).
    - Please follow our [Support Guidelines](https://fluxcd.io/support/)
      (in short: be nice, be respectful of volunteers' time, understand that maintainers and
      contributors cannot respond to all DMs, and keep discussions in the public #flux channel as much as possible).
- Have feature proposals or want to contribute?
    - Propose features on our [GitHub Discussions page](https://github.com/fluxcd/flux2/discussions).
    - Join our upcoming dev meetings ([meeting access and agenda](https://docs.google.com/document/d/1l_M0om0qUEN_NNiGgpqJ2tvsF2iioHkaARDeh6b70B0/view)).
    - [Join the flux-dev mailing list](https://lists.cncf.io/g/cncf-flux-dev).
    - Check out [how to contribute](CONTRIBUTING.md) to the project.
    - Check out the [project roadmap](https://fluxcd.io/roadmap/).

### Events

Check out our **[events calendar](https://fluxcd.io/#calendar)**,
both with upcoming talks, events and meetings you can attend.
Or view the **[resources section](https://fluxcd.io/resources)**
with past events videos you can watch.

We look forward to seeing you with us!
