/*
Copyright 2023 The Flux authors

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
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	customtypes "github.com/fluxcd/terraform-provider-flux/internal/framework/types"
	"github.com/fluxcd/terraform-provider-flux/internal/framework/validators"
)

const (
	defaultBranch = "main"
	defaultAuthor = "Flux"
)

type Ssh struct {
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	PrivateKey types.String `tfsdk:"private_key"`
}

type Http struct {
	Username             types.String `tfsdk:"username"`
	Password             types.String `tfsdk:"password"`
	InsecureHttpAllowed  types.Bool   `tfsdk:"allow_insecure_http"`
	CertificateAuthority types.String `tfsdk:"certificate_authority"`
}

type Git struct {
	Url                   customtypes.URL `tfsdk:"url"`
	Branch                types.String    `tfsdk:"branch"`
	AuthorName            types.String    `tfsdk:"author_name"`
	AuthorEmail           types.String    `tfsdk:"author_email"`
	GpgKeyRing            types.String    `tfsdk:"gpg_key_ring"`
	GpgPassphrase         types.String    `tfsdk:"gpg_passphrase"`
	GpgKeyID              types.String    `tfsdk:"gpg_key_id"`
	CommitMessageAppendix types.String    `tfsdk:"commit_message_appendix"`
	Ssh                   *Ssh            `tfsdk:"ssh"`
	Http                  *Http           `tfsdk:"http"`
}

type KubernetesExec struct {
	APIVersion types.String `tfsdk:"api_version"`
	Command    types.String `tfsdk:"command"`
	Env        types.Map    `tfsdk:"env"`
	Args       types.List   `tfsdk:"args"`
}

type Kubernetes struct {
	Host                  types.String    `tfsdk:"host"`
	Username              types.String    `tfsdk:"username"`
	Password              types.String    `tfsdk:"password"`
	Insecure              types.Bool      `tfsdk:"insecure"`
	ClientCertificate     types.String    `tfsdk:"client_certificate"`
	ClientKey             types.String    `tfsdk:"client_key"`
	ClusterCACertificate  types.String    `tfsdk:"cluster_ca_certificate"`
	ConfigPaths           types.Set       `tfsdk:"config_paths"`
	ConfigPath            types.String    `tfsdk:"config_path"`
	ConfigContext         types.String    `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String    `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String    `tfsdk:"config_context_cluster"`
	Token                 types.String    `tfsdk:"token"`
	ProxyURL              types.String    `tfsdk:"proxy_url"`
	Exec                  *KubernetesExec `tfsdk:"exec"`
}

type ProviderModel struct {
	Kubernetes *Kubernetes `tfsdk:"kubernetes"`
	Git        *Git        `tfsdk:"git"`
}

var _ provider.Provider = &fluxProvider{}
var _ provider.ProviderWithValidateConfig = &fluxProvider{}

type fluxProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &fluxProvider{
			version: version,
		}
	}
}

func (p *fluxProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "flux"
	resp.Version = p.version
}

func (p *fluxProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"kubernetes": schema.SingleNestedAttribute{
				Description: "Configuration block with settings for Kubernetes.",
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{
						Optional:    true,
						Description: "The hostname (in form of URI) of Kubernetes master.",
					},
					"username": schema.StringAttribute{
						Optional:    true,
						Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
					},
					"password": schema.StringAttribute{
						Optional:    true,
						Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
					},
					"insecure": schema.BoolAttribute{
						Optional:    true,
						Description: "Whether server should be accessed without verifying the TLS certificate.",
					},
					"client_certificate": schema.StringAttribute{
						Optional:    true,
						Description: "PEM-encoded client certificate for TLS authentication.",
					},
					"client_key": schema.StringAttribute{
						Optional:    true,
						Description: "PEM-encoded client certificate key for TLS authentication.",
					},
					"cluster_ca_certificate": schema.StringAttribute{
						Optional:    true,
						Description: "PEM-encoded root certificates bundle for TLS authentication.",
					},
					"config_paths": schema.SetAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "A list of paths to kube config files. Can be set with KUBE_CONFIG_PATHS environment variable.",
					},
					"config_path": schema.StringAttribute{
						Optional:    true,
						Description: "Path to the kube config file. Can be set with KUBE_CONFIG_PATH.",
					},
					"config_context": schema.StringAttribute{
						Optional:    true,
						Description: "Context to choose from the config file.",
					},
					"config_context_auth_info": schema.StringAttribute{
						Optional:    true,
						Description: "Authentication info context of the kube config (name of the kubeconfig user, `--user` flag in `kubectl`).",
					},
					"config_context_cluster": schema.StringAttribute{
						Optional:    true,
						Description: "Cluster context of the kube config (name of the kubeconfig cluster, `--cluster` flag in `kubectl`).",
					},
					"token": schema.StringAttribute{
						Optional:    true,
						Description: "Token to authenticate an service account.",
					},
					"proxy_url": schema.StringAttribute{
						Optional:    true,
						Description: "URL to the proxy to be used for all API requests.",
					},
					"exec": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"api_version": schema.StringAttribute{
								Description: "Kubernetes client authentication API Version.",
								Required:    true,
							},
							"command": schema.StringAttribute{
								Description: "Client authentication exec command.",
								Required:    true,
							},
							"env": schema.MapAttribute{
								ElementType: types.StringType,
								Description: "Client authentication exec environment variables.",
								Optional:    true,
							},
							"args": schema.ListAttribute{
								ElementType: types.StringType,
								Description: "Client authentication exec command arguments.",
								Optional:    true,
							},
						},
						Optional:    true,
						Description: "Kubernetes client authentication exec plugin configuration.",
					},
				},
				Optional: true,
			},
			"git": schema.SingleNestedAttribute{
				Description: "Configuration block with settings for Kubernetes.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						CustomType:  customtypes.URLType{},
						Description: "Url of git repository to bootstrap from.",
						Required:    true,
						Validators: []validator.String{
							validators.URLScheme("http", "https", "ssh"),
						},
					},
					"branch": schema.StringAttribute{
						Description: fmt.Sprintf("Branch in repository to reconcile from. Defaults to `%s`.", defaultBranch),
						Optional:    true,
					},
					"author_name": schema.StringAttribute{
						Description: fmt.Sprintf("Author name for Git commits. Defaults to `%s`.", defaultAuthor),
						Optional:    true,
					},
					"author_email": schema.StringAttribute{
						Description: "Author email for Git commits.",
						Optional:    true,
					},
					"gpg_key_ring": schema.StringAttribute{
						Description: "Path to the GPG key ring for signing commits.",
						Optional:    true,
					},
					"gpg_passphrase": schema.StringAttribute{
						Description: "Passphrase for decrypting GPG private key.",
						Optional:    true,
						Sensitive:   true,
					},
					"gpg_key_id": schema.StringAttribute{
						Description: "Key id for selecting a particular key.",
						Optional:    true,
					},
					"commit_message_appendix": schema.StringAttribute{
						Description: "String to add to the commit messages.",
						Optional:    true,
					},
					"ssh": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"username": schema.StringAttribute{
								Description: "Username for Git SSH server.",
								Optional:    true,
							},
							"password": schema.StringAttribute{
								Description: "Password for private key.",
								Optional:    true,
								Sensitive:   true,
							},
							"private_key": schema.StringAttribute{
								Description: "Private key used for authenticating to the Git SSH server.",
								Optional:    true,
								Sensitive:   true,
							},
						},
						Optional: true,
					},
					"http": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"username": schema.StringAttribute{
								Description: "Username for basic authentication.",
								Optional:    true,
							},
							"password": schema.StringAttribute{
								Description: "Password for basic authentication.",
								Optional:    true,
								Sensitive:   true,
							},
							"allow_insecure_http": schema.BoolAttribute{
								Description: "Allows http Git url connections.",
								Optional:    true,
							},
							"certificate_authority": schema.StringAttribute{
								Description: "Certificate authority to validate self-signed certificates.",
								Optional:    true,
							},
						},
						Optional: true,
					},
				},
				Optional: true,
			},
		},
	}
}

func (p *fluxProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var data ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Git != nil && data.Git.Url.ValueURL() != nil {
		if data.Git.Url.ValueURL().Scheme == "ssh" && data.Git.Http != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("git.http"),
				"Unexpected Attribute Configuration",
				"Did not expect http to be configured when url scheme is ssh",
			)
		}

		if (data.Git.Url.ValueURL().Scheme == "http" || data.Git.Url.ValueURL().Scheme == "https") && data.Git.Ssh != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("git.ssh"),
				"Unexpected Attribute Configuration",
				"Did not expect ssh to be configured when url scheme is http(s)",
			)
		}

		if data.Git.Url.ValueURL().Scheme == "http" && !data.Git.Http.InsecureHttpAllowed.ValueBool() {
			resp.Diagnostics.AddAttributeError(
				path.Root("git.allow_insecure_http"),
				"Scheme Validation Error",
				"Expected allow_insecure_http to be true when url scheme is http.",
			)
		}
	}
}

func (p *fluxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ProviderModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Either Git and Kubernetes configuration is set or none of them are set.
	// An error is returned if either or is set.
	if data.Git == nil && data.Kubernetes == nil {
		return
	}
	if data.Git == nil && data.Kubernetes != nil {
		resp.Diagnostics.AddError("Git configuration is empty when Kubernetes is not", "Either none or both Git and Kubernetes blocks need to be set")
		return
	}
	if data.Git != nil && data.Kubernetes == nil {
		resp.Diagnostics.AddError("Kubernetes configuration is empty when Git is not", "Either none or both Git and Kubernetes blocks need to be set")
		return
	}

	// Set default values.
	if data.Git.Branch.IsNull() {
		data.Git.Branch = types.StringValue(defaultBranch)
	}
	if data.Git.AuthorName.IsNull() {
		data.Git.AuthorName = types.StringValue(defaultAuthor)
	}
	if data.Kubernetes.ConfigPath.IsNull() {
		if v, ok := os.LookupEnv("KUBE_CONFIG_PATH"); ok {
			data.Kubernetes.ConfigPath = types.StringValue(v)
		}
	}
	if data.Kubernetes.ConfigPaths.IsNull() {
		if v, ok := os.LookupEnv("KUBE_CONFIG_PATHS"); ok {
			var paths []attr.Value
			for _, p := range filepath.SplitList(v) {
				paths = append(paths, types.StringValue(p))
			}
			data.Kubernetes.ConfigPaths = types.SetValueMust(types.StringType, paths)
		}
	}

	prd, err := NewProviderResourceData(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Could not create provider resource data", err.Error())
		return
	}
	resp.ResourceData = prd
}

func (p *fluxProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *fluxProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBootstrapGitResource,
	}
}
