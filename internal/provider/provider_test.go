// Copyright (c) The Flux authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"flux": providerserver.NewProtocol6WithError(New("dev")()),
}
