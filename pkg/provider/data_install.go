package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/fluxcd/flux2/pkg/manifestgen/install"
)

var (
	installDefaults = install.MakeDefaultOptions()
)

func DataInstall() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataInstallRead,
		Schema: map[string]*schema.Schema{
			"target_path": {
				Type:     schema.TypeString,
				Required: true,
			},
			"base_url": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  installDefaults.BaseURL,
			},
			"version": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  installDefaults.Version,
			},
			"namespace": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  installDefaults.Namespace,
			},
			"components": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"registry": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  installDefaults.Registry,
			},
			"image_pull_secrets": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  installDefaults.ImagePullSecret,
			},
			"arch": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  installDefaults.Arch,
			},
			"watch_all_namespaces": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  installDefaults.WatchAllNamespaces,
			},
			"network_policy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  installDefaults.NetworkPolicy,
			},
			"log_level": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  installDefaults.LogLevel,
			},
			"timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  installDefaults.Timeout,
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

func dataInstallRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	components := toStringList(d.Get("components"))
	if len(components) == 0 {
		components = installDefaults.Components
	}

	opt := install.MakeDefaultOptions()
	opt.BaseURL = d.Get("base_url").(string)
	opt.Version = d.Get("version").(string)
	opt.Namespace = d.Get("namespace").(string)
	opt.Components = components
	opt.Registry = d.Get("registry").(string)
	opt.ImagePullSecret = d.Get("image_pull_secrets").(string)
	opt.Arch = d.Get("arch").(string)
	opt.WatchAllNamespaces = d.Get("watch_all_namespaces").(bool)
	opt.NetworkPolicy = d.Get("network_policy").(bool)
	opt.LogLevel = d.Get("log_level").(string)
	timeout := d.Get("timeout").(int)
	opt.Timeout = time.Duration(timeout)
	opt.TargetPath = d.Get("target_path").(string)
	manifest, err := install.Generate(opt)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%x", sha256.Sum256([]byte(manifest.Path+manifest.Content))))
	d.Set("path", manifest.Path)
	d.Set("content", manifest.Content)

	return nil
}

func toStringList(ll interface{}) []string {
	set := ll.(*schema.Set)
	result := []string{}
	for _, l := range set.List() {
		result = append(result, l.(string))
	}

	return result
}
