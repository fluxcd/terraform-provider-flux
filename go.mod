module github.com/fluxcd/terraform-provider-flux

go 1.15

require (
	github.com/fluxcd/flux2 v0.7.7
	github.com/hashicorp/terraform-plugin-docs v0.3.1
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.3.0
	github.com/stretchr/testify v1.6.1
)

replace github.com/fluxcd/flux2 => github.com/fluxcd/flux2 v0.7.8-0.20210211132019-37f558708523
