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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/pkg/manifestgen/sync"

	"github.com/fluxcd/terraform-provider-flux/internal/framework/validators"
	"github.com/fluxcd/terraform-provider-flux/internal/utils"
)

type syncDataSourceData struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Namespace        types.String `tfsdk:"namespace"`
	URL              types.String `tfsdk:"url"`
	Branch           types.String `tfsdk:"branch"`
	Tag              types.String `tfsdk:"tag"`
	Semver           types.String `tfsdk:"semver"`
	Commit           types.String `tfsdk:"commit"`
	Secret           types.String `tfsdk:"secret"`
	TargetPath       types.String `tfsdk:"target_path"`
	Interval         types.Int64  `tfsdk:"interval"`
	PatchNames       types.Set    `tfsdk:"patch_names"`
	Path             types.String `tfsdk:"path"`
	Content          types.String `tfsdk:"content"`
	KustomizePath    types.String `tfsdk:"kustomize_path"`
	KustomizeContent types.String `tfsdk:"kustomize_content"`
	PatchFilePaths   types.Map    `tfsdk:"patch_file_paths"`
}

var _ datasource.DataSource = &syncDataSource{}

type syncDataSource struct{}

func NewSyncDataSource() datasource.DataSource {
	return &syncDataSource{}
}

func (s *syncDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sync"
}

func (s *syncDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	opts := sync.MakeDefaultOptions()
	resp.Schema = schema.Schema{
		MarkdownDescription: "`flux_sync` can be used to generate manifests for reconciling the specified repository path on the cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Description: fmt.Sprintf("The kubernetes resources name. Defaults to `%s`.", opts.Name),
				Optional:    true,
			},
			"namespace": schema.StringAttribute{
				Description: fmt.Sprintf("The namespace scope for this operation. Defaults to `%s`.", opts.Namespace),
				Optional:    true,
			},
			"url": schema.StringAttribute{
				Description: "Git repository clone url.",
				Required:    true,
				Validators: []validator.String{
					validators.URLScheme("https", "http", "ssh"),
				},
			},
			"branch": schema.StringAttribute{
				Description: fmt.Sprintf("Default branch to sync from. Defaults to `%s`.", opts.Branch),
				Optional:    true,
			},
			"tag": schema.StringAttribute{
				Description: "The Git tag to checkout, takes precedence over `branch`.",
				Optional:    true,
			},
			"semver": schema.StringAttribute{
				Description: "The Git tag semver expression, takes precedence over `tag`.",
				Optional:    true,
			},
			"commit": schema.StringAttribute{
				Description: "The Git commit SHA to checkout, if specified Tag filters will be ignored.",
				Optional:    true,
			},
			"secret": schema.StringAttribute{
				Description: fmt.Sprintf("The name of the secret that is referenced by GitRepository as SecretRef. Defaults to `%s`.", opts.Secret),
				Optional:    true,
			},
			"target_path": schema.StringAttribute{
				Description: "Relative path to the Git repository root where the sync manifests are committed.",
				Required:    true,
			},
			"interval": schema.Int64Attribute{
				Description: "Sync interval in minutes. Defaults to `1`.",
				Optional:    true,
			},
			"patch_names": schema.SetAttribute{
				Description: "The names of patches to apply to the Kustomization. Used to generate the `patch_file_paths` output value.",
				ElementType: types.StringType,
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
			"kustomize_path": schema.StringAttribute{
				Description: "Expected path of kustomize content in git repository.",
				Computed:    true,
			},
			"kustomize_content": schema.StringAttribute{
				Description: "Kustomize yaml document.",
				Computed:    true,
			},
			"patch_file_paths": schema.MapAttribute{
				Description: "Map of expected paths of kustomize patches in git repository, keyed by the `patch_names` input variable.",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (s *syncDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data syncDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	opt := sync.MakeDefaultOptions()

	// Set default values to data object
	if data.Interval.IsNull() {
		data.Interval = types.Int64Value(1)
	}
	if data.Name.IsNull() {
		data.Name = types.StringValue(opt.Name)
	}
	if data.Namespace.IsNull() {
		data.Namespace = types.StringValue(opt.Namespace)
	}
	if data.URL.IsNull() {
		data.URL = types.StringValue(opt.URL)
	}
	if data.Branch.IsNull() {
		data.Branch = types.StringValue(opt.Branch)
	}
	if data.Secret.IsNull() {
		data.Secret = types.StringValue(opt.Secret)
	}

	// Set data values to option
	opt.TargetPath = data.TargetPath.ValueString()
	opt.Interval = time.Duration(data.Interval.ValueInt64()) * time.Minute
	opt.Name = data.Name.ValueString()
	opt.Namespace = data.Namespace.ValueString()
	opt.URL = data.URL.ValueString()
	opt.Branch = data.Branch.ValueString()
	opt.Secret = data.Secret.ValueString()
	opt.Tag = data.Tag.ValueString()
	opt.SemVer = data.Semver.ValueString()
	opt.Commit = data.Commit.ValueString()

	// Compute manifests
	manifest, err := sync.Generate(opt)
	if err != nil {
		resp.Diagnostics.AddError("could not generate sync manifets", err.Error())
		return
	}
	basePath := path.Dir(manifest.Path)
	kustomizePath := path.Join(basePath, "kustomization.yaml")
	paths := []string{opt.ManifestFile, install.MakeDefaultOptions().ManifestFile}
	patchNames := []string{}
	diags = data.PatchNames.ElementsAs(ctx, &patchNames, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	patchBases := utils.GetPatchBases(patchNames)
	kustomizeContent, err := utils.GenerateKustomizationYaml(paths, patchBases)
	if err != nil {
		resp.Diagnostics.AddError("could not generate kustomization manifets", err.Error())
		return
	}
	patchFilePathsMap := utils.GenPatchFilePaths(basePath, patchNames)

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%x", sha256.Sum256([]byte(manifest.Path+manifest.Content))))
	data.Path = types.StringValue(manifest.Path)
	data.Content = types.StringValue(manifest.Content)
	data.KustomizePath = types.StringValue(kustomizePath)
	data.KustomizeContent = types.StringValue(kustomizeContent)
	val, diags := types.MapValueFrom(ctx, types.StringType, patchFilePathsMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.PatchFilePaths = val

	// Write to state
	diags = resp.State.Set(ctx, data)
	resp.Diagnostics.Append(diags...)
}
