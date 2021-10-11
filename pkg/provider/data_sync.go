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
	"path"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/pkg/manifestgen/sync"
)

var (
	syncDefaults = sync.MakeDefaultOptions()
)

func DataSync() *schema.Resource {
	return &schema.Resource{
		Description: "`flux_sync` can be used to generate manifests for reconciling the specified repository path on the cluster.",
		ReadContext: dataSyncRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "The kubernetes resources name",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     syncDefaults.Namespace,
			},
			"namespace": {
				Description: "The namespace scope for this operation.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     syncDefaults.Namespace,
			},
			"url": {
				Description:  "Git repository clone url.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsURLWithScheme([]string{"http", "https", "ssh"}),
			},
			"branch": {
				Description: "Default branch to sync from.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     syncDefaults.Branch,
			},
			"tag": {
				Description: "The Git tag to checkout, takes precedence over `branch`.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"semver": {
				Description: "The Git tag semver expression, takes precedence over `tag`.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"commit": {
				Description: "The Git commit SHA to checkout, if specified Tag filters will be ignored.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"secret": {
				Description: "The name of the secret that is referenced by GitRepository as SecretRef.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     syncDefaults.Secret,
			},
			"target_path": {
				Description: "Relative path to the Git repository root where the sync manifests are committed.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"interval": {
				Description: "Sync interval in minutes.",
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     fmt.Sprintf("%d", int64(syncDefaults.Interval.Minutes())),
			},
			"git_implementation": {
				Description: "The git implementation to use, can be `go-git` or `libgit2`.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     syncDefaults.GitImplementation,
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
			"kustomize_path": {
				Description: "Expected path of kustomize content in git repository.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"kustomize_content": {
				Description: "Kustomize yaml document.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSyncRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	opt := sync.MakeDefaultOptions()
	interval := d.Get("interval").(int)
	opt.Interval = time.Duration(interval) * time.Minute
	opt.URL = d.Get("url").(string)
	opt.Name = d.Get("name").(string)
	opt.Namespace = d.Get("namespace").(string)
	opt.Branch = d.Get("branch").(string)
	opt.TargetPath = d.Get("target_path").(string)
	opt.GitImplementation = d.Get("git_implementation").(string)
	opt.Secret = d.Get("secret").(string)

	if v, ok := d.GetOk("tag"); ok {
		opt.Tag = v.(string)
	}

	if v, ok := d.GetOk("semver"); ok {
		opt.SemVer = v.(string)
	}

	if v, ok := d.GetOk("commit"); ok {
		opt.Commit = v.(string)
	}

	manifest, err := sync.Generate(opt)
	if err != nil {
		return diag.FromErr(err)
	}

	basePath := path.Dir(manifest.Path)
	kustomizePath := path.Join(basePath, "kustomization.yaml")
	paths := []string{opt.ManifestFile, install.MakeDefaultOptions().ManifestFile}
	kustomizeContent, err := generateKustomizationYaml(paths)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(manifest.Path+manifest.Content))))
	d.Set("path", manifest.Path)
	d.Set("content", manifest.Content)
	d.Set("kustomize_path", kustomizePath)
	d.Set("kustomize_content", kustomizeContent)

	return nil
}
