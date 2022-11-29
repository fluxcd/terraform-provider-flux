/*
Copyright 2020 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"

	"github.com/fluxcd/terraform-provider-flux/internal/provider"
	legacyprovider "github.com/fluxcd/terraform-provider-flux/pkg/provider"
)

var (
	version string = "dev"
)

func main() {
	debugFlag := flag.Bool("debug", false, "Start provider in debug mode.")
	flag.Parse()
	ctx := context.Background()

	upgradedSdkProvider, err := tf5to6server.UpgradeServer(
		ctx,
		legacyprovider.Provider().GRPCProvider,
	)
	if err != nil {
		log.Fatal(err)
	}
	providers := []func() tfprotov6.ProviderServer{
		providerserver.NewProtocol6(provider.New(version)()),
		func() tfprotov6.ProviderServer {
			return upgradedSdkProvider
		},
	}
	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		log.Fatal(err)
	}
	var serveOpts []tf6server.ServeOpt
	if *debugFlag {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}
	err = tf6server.Serve(
		"registry.terraform.io/fluxcd/flux",
		muxServer.ProviderServer,
		serveOpts...,
	)
	if err != nil {
		log.Fatal(err)
	}
}
