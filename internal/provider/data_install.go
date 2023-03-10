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
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/fluxcd/terraform-provider-flux/internal/utils"
)

type installDataSourceData struct {
	ID                 types.String `tfsdk:"id"`
	TargetPath         types.String `tfsdk:"target_path"`
	Version            types.String `tfsdk:"version"`
	Namespace          types.String `tfsdk:"namespace"`
	ClusterDomain      types.String `tfsdk:"cluster_domain"`
	Components         types.Set    `tfsdk:"components"`
	ComponentsExtra    types.Set    `tfsdk:"components_extra"`
	Registry           types.String `tfsdk:"registry"`
	ImagePullSecrets   types.String `tfsdk:"image_pull_secrets"`
	WatchAllNamespaces types.Bool   `tfsdk:"watch_all_namespaces"`
	NetworkPolicy      types.Bool   `tfsdk:"network_policy"`
	LogLevel           types.String `tfsdk:"log_level"`
	TolerationKeys     types.Set    `tfsdk:"toleration_keys"`
	BaseURL            types.String `tfsdk:"baseurl"`
	Path               types.String `tfsdk:"path"`
	Content            types.String `tfsdk:"content"`
}

var _ datasource.DataSource = &installDataSource{}

type installDataSource struct{}

func NewInstallDataSource() datasource.DataSource {
	return &installDataSource{}
}

func (s *installDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_install"
}

func (s *installDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	opts := install.MakeDefaultOptions()
	resp.Schema = schema.Schema{
		MarkdownDescription: "`flux_install` can be used to generate Kubernetes manifests for deploying Flux.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"target_path": schema.StringAttribute{
				Description: "Relative path to the Git repository root where Flux manifests are committed.",
				Required:    true,
			},
			"version": schema.StringAttribute{
				Description: fmt.Sprintf("Flux version. Defaults to `%s`.", utils.DefaultFluxVersion),
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("(latest|^v.*)"), "must either be latest or start with 'v'"),
				},
			},
			"namespace": schema.StringAttribute{
				Description: fmt.Sprintf("The namespace scope for install manifests. Defaults to `%s`.", opts.Namespace),
				Optional:    true,
			},
			"cluster_domain": schema.StringAttribute{
				Description: fmt.Sprintf("The internal cluster domain. Defaults to `%s`.", opts.ClusterDomain),
				Optional:    true,
			},
			"components": schema.SetAttribute{
				Description: "Toolkit components to include in the install manifests.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"components_extra": schema.SetAttribute{
				Description: "List of extra components to include in the install manifests.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"registry": schema.StringAttribute{
				Description: fmt.Sprintf("Container registry where the toolkit images are published. Defaults to `%s`.", opts.Registry),
				Optional:    true,
			},
			"image_pull_secrets": schema.StringAttribute{
				Description: "Kubernetes secret name used for pulling the toolkit images from a private registry.",
				Optional:    true,
			},
			"watch_all_namespaces": schema.BoolAttribute{
				Description: fmt.Sprintf("If true watch for custom resources in all namespaces. Defaults to `%v`.", opts.WatchAllNamespaces),
				Optional:    true,
			},
			"network_policy": schema.BoolAttribute{
				Description: fmt.Sprintf("Deny ingress access to the toolkit controllers from other namespaces using network policies. Defaults to `%v`.", opts.NetworkPolicy),
				Optional:    true,
			},
			"log_level": schema.StringAttribute{
				Description: fmt.Sprintf("Log level for toolkit components. Defaults to `%s`.", opts.LogLevel),
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("info", "debug", "error"),
				},
			},
			"toleration_keys": schema.SetAttribute{
				Description: "List of toleration keys used to schedule the components pods onto nodes with matching taints.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"baseurl": schema.StringAttribute{
				Description: fmt.Sprintf("Base URL to get the install manifests from. When specifying this, `version` should also be set to the corresponding version to download from that URL, or the latest version associated with upstream flux will be requested. Defaults to `%s`.", opts.BaseURL),
				Optional:    true,
			},
			"path": schema.StringAttribute{
				Description: "Expected path of content in git repository.",
				Computed:    true,
			},
			"content": schema.StringAttribute{
				Description: "Manifests in multi-doc yaml format.",
				Computed:    true,
			},
		},
	}
}

func (s *installDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data installDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	opt := install.MakeDefaultOptions()

	// Set default values to data object
	if data.Components.IsNull() {
		v, diags := types.SetValueFrom(ctx, types.StringType, opt.Components)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Components = v
	}
	if data.Version.IsNull() {
		data.Version = types.StringValue(utils.DefaultFluxVersion)
	}
	if data.Namespace.IsNull() {
		data.Namespace = types.StringValue(opt.Namespace)
	}
	if data.ClusterDomain.IsNull() {
		data.ClusterDomain = types.StringValue(opt.ClusterDomain)
	}
	if data.Registry.IsNull() {
		data.Registry = types.StringValue(opt.Registry)
	}
	if data.ImagePullSecrets.IsNull() {
		data.ImagePullSecrets = types.StringValue(opt.ImagePullSecret)
	}
	if data.NetworkPolicy.IsNull() {
		data.NetworkPolicy = types.BoolValue(opt.NetworkPolicy)
	}
	if data.WatchAllNamespaces.IsNull() {
		data.WatchAllNamespaces = types.BoolValue(opt.WatchAllNamespaces)
	}
	if data.LogLevel.IsNull() {
		data.LogLevel = types.StringValue(opt.LogLevel)
	}
	if data.BaseURL.IsNull() {
		data.BaseURL = types.StringValue(opt.BaseURL)
	}

	// Set data values to option
	opt.TargetPath = data.TargetPath.ValueString()
	components := []string{}
	diags = data.Components.ElementsAs(ctx, &components, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	componentsExtra := []string{}
	diags = data.ComponentsExtra.ElementsAs(ctx, &componentsExtra, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	components = append(components, componentsExtra...)
	opt.Components = components
	opt.ComponentsExtra = componentsExtra
	tolerationKeys := []string{}
	diags = data.TolerationKeys.ElementsAs(ctx, &tolerationKeys, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(tolerationKeys) != 0 {
		opt.TolerationKeys = tolerationKeys
	}
	opt.Version = data.Version.ValueString()
	opt.Namespace = data.Namespace.ValueString()
	opt.ClusterDomain = data.ClusterDomain.ValueString()
	opt.Registry = data.Registry.ValueString()
	opt.ImagePullSecret = data.ImagePullSecrets.ValueString()
	opt.NetworkPolicy = data.NetworkPolicy.ValueBool()
	opt.WatchAllNamespaces = data.WatchAllNamespaces.ValueBool()
	opt.LogLevel = data.LogLevel.ValueString()
	opt.BaseURL = data.BaseURL.ValueString()

	// Compute manifests
	manifest, err := install.Generate(opt, "")
	if err != nil {
		resp.Diagnostics.AddError("coudl not generate manifests", err.Error())
		return
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%x", sha256.Sum256([]byte(manifest.Path+manifest.Content))))
	data.Path = types.StringValue(manifest.Path)
	data.Content = types.StringValue(manifest.Content)

	// Write to state
	diags = resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diags...)
}
