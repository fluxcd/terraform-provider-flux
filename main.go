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
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"

	"github.com/fluxcd/flux2/v2/pkg/manifestgen"
	"github.com/go-logr/logr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/fluxcd/terraform-provider-flux/internal/provider"
)

var (
	version string = "dev"
)

//go:embed manifests/*.yaml
var embeddedManifests embed.FS

func main() {
	ctrllog.SetLogger(logr.New(ctrllog.NullLogSink{}))
	debugFlag := flag.Bool("debug", false, "Start provider in debug mode.")
	flag.Parse()

	// extract the embedded Flux manifests to tmp directory
	tmpBaseDir, err := writeEmbeddedManifests()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer os.RemoveAll(tmpBaseDir)
	provider.EmbeddedManifests = tmpBaseDir

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/fluxcd/flux",
		Debug:   *debugFlag,
	}
	err = providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func writeEmbeddedManifests() (string, error) {
	tmpBaseDir, err := manifestgen.MkdirTempAbs("", "flux-manifests-")
	if err != nil {
		return "", err
	}

	manifests, err := fs.ReadDir(embeddedManifests, "manifests")
	if err != nil {
		return tmpBaseDir, err
	}
	for _, manifest := range manifests {
		data, err := fs.ReadFile(embeddedManifests, path.Join("manifests", manifest.Name()))
		if err != nil {
			return tmpBaseDir, fmt.Errorf("reading file failed: %w", err)
		}

		err = os.WriteFile(path.Join(tmpBaseDir, manifest.Name()), data, 0666)
		if err != nil {
			return tmpBaseDir, fmt.Errorf("writing file failed: %w", err)
		}
	}

	return tmpBaseDir, nil
}
