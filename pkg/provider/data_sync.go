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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/fluxcd/flux2/pkg/manifestgen/sync"
)

var (
	syncDefaults = sync.MakeDefaultOptions()
)

func DataSync() *schema.Resource {
	return &schema.Resource{
		Description: "`flux_sync` Returns the manifest for a gitsource configuration for flux ",

		ReadContext: dataSyncRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  syncDefaults.Name,
			},
			"namespace": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  syncDefaults.Namespace,
			},
			"url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"branch": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  syncDefaults.Branch,
			},
			"target_path": {
				Type:     schema.TypeString,
				Required: true,
			},
			"interval": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  syncDefaults.Interval,
			},
			"path": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"content": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSyncRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	opt := sync.MakeDefaultOptions()
	interval := d.Get("interval").(int)
	opt.Interval = time.Duration(interval)
	opt.URL = d.Get("url").(string)
	opt.Name = d.Get("name").(string)
	opt.Namespace = d.Get("namespace").(string)
	opt.Branch = d.Get("branch").(string)
	opt.TargetPath = d.Get("target_path").(string)
	manifest, err := sync.Generate(opt)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(manifest.Path+manifest.Content))))
	d.Set("path", manifest.Path)
	d.Set("content", manifest.Content)

	return nil
}
