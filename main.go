package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/fluxcd/terraform-provider-flux/pkg/provider"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.Provider,
	})
}
