# Contributing

The Flux Terraform provider is [Apache 2.0
licensed](https://github.com/fluxcd/flux2/blob/main/LICENSE) and
accepts contributions via GitHub pull requests. This document outlines
some of the conventions on to make it easier to get your contribution
accepted.

We gratefully welcome improvements to issues and documentation as well as to
code.

## Certificate of Origin

By contributing to the Flux project you agree to the Developer Certificate of
Origin (DCO). This document was created by the Linux Kernel community and is a
simple statement that you, as a contributor, have the legal right to make the
contribution.

We require all commits to be signed. By signing off with your signature, you
certify that you wrote the patch or otherwise have the right to contribute the
material by the rules of the [DCO](DCO):

`Signed-off-by: Jane Doe <jane.doe@example.com>`

The signature must contain your real name
(sorry, no pseudonyms or anonymous contributions)
If your `user.name` and `user.email` are configured in your Git config,
you can sign your commit automatically with `git commit -s`.

## Communications

For realtime communications we use Slack: To join the conversation, simply
join the [CNCF](https://slack.cncf.io/) Slack workspace and use the
[#flux-dev](https://cloud-native.slack.com/messages/flux-dev/) channel.

To discuss ideas and specifications we use [Github
Discussions](https://github.com/fluxcd/flux2/discussions).

For announcements we use a mailing list as well. Simply subscribe to
[flux-dev on cncf.io](https://lists.cncf.io/g/cncf-flux-dev)
to join the conversation (there you can also add calendar invites
to your Google calendar for our [Flux
meeting](https://docs.google.com/document/d/1l_M0om0qUEN_NNiGgpqJ2tvsF2iioHkaARDeh6b70B0/view)).

## Developing

Clone the terraform-provider-flux repository.

```sh
git clone https://github.com/fluxcd/terraform-provider-flux.git
cd terraform-provider-flux
```

Run the unit and acceptance tests.

```bash
make testacc
```

Generate the docs if you have made changes to any of the schemas or guides.

```sh
make docs
```

First build the provider, then generate a Terraform CLI config file to use a local build of the provider with Terraform.

```sh
make build
make terraformrc
export TF_CLI_CONFIG_FILE="${PWD}/.terraformrc"
```


## Documentation

The documentation is generated from the `*.md.tmpl` files in `templates/` with
[tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs).

To generate the documentation, run:

```sh
make docs
```

Documentation is written in Markdown format and supports
[frontmatter](https://www.terraform.io/docs/registry/providers/docs.html#yaml-frontmatter),
which should be used to ensure [navigation hierarchy](https://www.terraform.io/docs/registry/providers/docs.html#navigation-hierarchy).
[(Markdown) format](https://www.terraform.io/docs/registry/providers/docs.html#format)
recommendations should be followed.

New pages can be added by adding a new `.md.tmpl` file to the
`templates/` (sub-)directory. Subcategories have an [impact on
directory structure](https://www.terraform.io/docs/registry/providers/docs.html#guides-subcategories),
other path related assumptions are [documented here](https://github.com/hashicorp/terraform-plugin-docs#conventional-paths).

## Acceptance policy

These things will make a PR more likely to be accepted:

- a well-described requirement
- sign-off all your commits
- tests for new code
- tests for old code!
- new code and tests follow the conventions in old code and tests
- a good commit message (see below)
- all code must abide [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- names should abide [What's in a name](https://talks.golang.org/2014/names.slide#1)
- code must build on both Linux and Darwin, via plain `go build`
- code should have appropriate test coverage and tests should be written
  to work with `go test`

In general, we will merge a PR once one maintainer has endorsed it.
For substantial changes, more people may become involved, and you might
get asked to resubmit the PR or divide the changes into more than one PR.

### Format of the Commit Message

We prefer the following rules for good commit messages:

- Limit the subject to 50 characters and write as the continuation
  of the sentence "If applied, this commit will ..."
- Explain what and why in the body, if more than a trivial change;
  wrap it at 72 characters.

The [following article](https://chris.beams.io/posts/git-commit/#seven-rules)
has some more helpful advice on documenting your work.
