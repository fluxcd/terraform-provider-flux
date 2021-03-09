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

package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/fluxcd/flux2/pkg/manifestgen/install"
)

var (
	installDefaults = install.MakeDefaultOptions()
)

func DataInstall() *schema.Resource {
	return &schema.Resource{
		Description: "`flux_install` can be used to generate Kubernetes manifests for deploying Flux.",
		ReadContext: dataInstallRead,
		Schema: map[string]*schema.Schema{
			"target_path": {
				Description: "Relative path to the Git repository root where Flux manifests are committed.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"version": {
				Description: "Flux version.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "v0.9.1",
				ValidateFunc: func(val interface{}, key string) ([]string, []error) {
					errs := []error{}
					v := val.(string)
					if v != "latest" && !strings.HasPrefix(v, "v") {
						errs = append(errs, fmt.Errorf("%q must either be latest or have the prefix 'v', got: %s", key, v))
					}
					return []string{}, errs
				},
			},
			"namespace": {
				Description: "The namespace scope for install manifests.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     installDefaults.Namespace,
			},
			"cluster_domain": {
				Description: "The internal cluster domain.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     installDefaults.ClusterDomain,
			},
			"components": {
				Description: "Toolkit components to include in the install manifests.",
				Type:        schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"registry": {
				Description: "Container registry where the toolkit images are published.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     installDefaults.Registry,
			},
			"image_pull_secrets": {
				Description: "Kubernetes secret name used for pulling the toolkit images from a private registry.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     installDefaults.ImagePullSecret,
			},
			"watch_all_namespaces": {
				Description: "If true watch for custom resources in all namespaces.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     installDefaults.WatchAllNamespaces,
			},
			"network_policy": {
				Description: "Deny ingress access to the toolkit controllers from other namespaces using network policies.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     installDefaults.NetworkPolicy,
			},
			"log_level": {
				Description:  "Log level for toolkit components.",
				Type:         schema.TypeString,
				Optional:     true,
				Default:      installDefaults.LogLevel,
				ValidateFunc: validation.StringInSlice([]string{"info", "debug", "error"}, false),
			},
			"toleration_keys": {
				Description: "List of toleration keys used to schedule the components pods onto nodes with matching taints.",
				Type:        schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"path": {
				Description: "Expected path of content in git repository.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"content": {
				Description: "Manifests in multi-doc yaml format.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataInstallRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	components := toStringList(d.Get("components"))
	if len(components) == 0 {
		components = installDefaults.Components
	}

	tolerationKeys := toStringList(d.Get("toleration_keys"))
	if len(tolerationKeys) == 0 {
		tolerationKeys = installDefaults.TolerationKeys
	}

	opt := install.MakeDefaultOptions()
	opt.Version = d.Get("version").(string)
	opt.Namespace = d.Get("namespace").(string)
	opt.ClusterDomain = d.Get("cluster_domain").(string)
	opt.Components = components
	opt.Registry = d.Get("registry").(string)
	opt.ImagePullSecret = d.Get("image_pull_secrets").(string)
	opt.WatchAllNamespaces = d.Get("watch_all_namespaces").(bool)
	opt.NetworkPolicy = d.Get("network_policy").(bool)
	opt.LogLevel = d.Get("log_level").(string)
	opt.TargetPath = d.Get("target_path").(string)
	opt.TolerationKeys = tolerationKeys
	manifest, err := install.Generate(opt, "")
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(manifest.Path+manifest.Content))))
	d.Set("path", manifest.Path)
	d.Set("content", manifest.Content)

	return nil
}
