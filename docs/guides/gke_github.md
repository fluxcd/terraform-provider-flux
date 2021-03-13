---
subcategory: ""
page_title: "Bootstrap flux on GKE with GitHub"
description: |-
  An example of how to bootstrap flux on GKE with a GitHub repository.
---

# Bootstrap flux on GKE with GitHub

In order to follow the guide you'll need a GitHub account and a [personal access token](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token)
that can create repositories (check all permissions under repo) and GKE.

You can set these in the environments that your terraform runs.

```shell
TF_VAR_github_token=<token>
```
