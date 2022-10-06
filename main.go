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
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/hashicorp/terraform-plugin-mux/tf6to5server"

	"github.com/fluxcd/terraform-provider-flux/internal/provider"
	legacyprovider "github.com/fluxcd/terraform-provider-flux/pkg/provider"
)

var (
	// Version can be updated by goreleaser on release
	version string = "dev"
)

func main() {
	debugFlag := flag.Bool("debug", false, "Start provider in debug mode.")
	flag.Parse()

	// TODO: Verify that the downgrade is correctly configured
	ctx := context.Background()
	downgradedFrameworkProvider, err := tf6to5server.DowngradeServer(
		ctx,
		providerserver.NewProtocol6(provider.New(version)()),
	)
	providers := []func() tfprotov5.ProviderServer{
		func() tfprotov5.ProviderServer {
			return downgradedFrameworkProvider
		},
		legacyprovider.Provider().GRPCProvider,
	}

	muxServer, err := tf5muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf5server.ServeOpt
	if *debugFlag {
		serveOpts = append(serveOpts, tf5server.WithManagedDebug())
	}
	err = tf5server.Serve(
		"registry.terraform.io/providers/fluxcd/flux",
		muxServer.ProviderServer,
		serveOpts...,
	)
	if err != nil {
		log.Fatal(err)
	}
}
