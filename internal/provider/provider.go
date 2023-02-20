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
	"bytes"
	"context"
	"fmt"

	runclient "github.com/fluxcd/pkg/runtime/client"
	"github.com/fluxcd/pkg/ssa"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mitchellh/go-homedir"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/fluxcd/terraform-provider-flux/internal/utils"
)

type providerResourceData struct {
	rcg        *utils.RESTClientGetter
	manager    *ssa.ResourceManager
	kubeClient client.WithWatch
}

func (prd *providerResourceData) HasClients() bool {
	return prd.rcg != nil && prd.manager != nil && prd.kubeClient != nil
}

type providerData struct {
	Host                  types.String `tfsdk:"host"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	Insecure              types.Bool   `tfsdk:"insecure"`
	ClientCertificate     types.String `tfsdk:"client_certificate"`
	ClientKey             types.String `tfsdk:"client_key"`
	ClusterCACertificate  types.String `tfsdk:"cluster_ca_certificate"`
	ConfigPaths           types.Set    `tfsdk:"config_paths"`
	ConfigPath            types.String `tfsdk:"config_path"`
	ConfigContext         types.String `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String `tfsdk:"config_context_cluster"`
	Token                 types.String `tfsdk:"token"`
	ProxyURL              types.String `tfsdk:"proxy_url"`
}

var _ provider.Provider = &fluxProvider{}

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
				Optional: true,
			},
			"config_context_auth_info": schema.StringAttribute{
				Optional:    true,
				Description: "",
			},
			"config_context_cluster": schema.StringAttribute{
				Optional:    true,
				Description: "",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Description: "Token to authenticate an service account",
			},
			"proxy_url": schema.StringAttribute{
				Optional:    true,
				Description: "URL to the proxy to be used for all API requests",
			},
		},
	}
}

func (p *fluxProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	clientCfg, err := initializeConfiguration(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to initialize configuration", err.Error())
		return
	}
	// We need to ignore this error as configure will also be called when using the old provider.
	// If we enforce full provider configuration the old provider will not work.
	// TODO: Remove this when old provider is removed.
	if _, err := clientCfg.ClientConfig(); err != nil {
		return
	}
	rcg := utils.NewRestClientGetter(clientCfg)
	man, err := utils.ResourceManager(rcg, &runclient.Options{})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Kubernetes Resource Manager", err.Error())
		return
	}
	kubeClient, err := utils.KubeClient(rcg, &runclient.Options{})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Kubernetes Client", err.Error())
		return
	}
	resp.ResourceData = &providerResourceData{
		rcg:        rcg,
		manager:    man,
		kubeClient: kubeClient,
	}
}

func (p *fluxProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}

func (p *fluxProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBootstrapGitResource,
	}
}

func initializeConfiguration(ctx context.Context, data providerData) (clientcmd.ClientConfig, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}
	if data.ConfigPath.ValueString() != "" {
		configPaths = []string{data.ConfigPath.ValueString()}
	} else if len(data.ConfigPaths.Elements()) > 0 {
		var pp []string
		diag := data.ConfigPaths.ElementsAs(ctx, &pp, false)
		if diag.HasError() {
			return nil, fmt.Errorf("%s", diag)
		}
		for _, p := range pp {
			configPaths = append(configPaths, p)
		}
	}
	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				return nil, err
			}
			expandedPaths = append(expandedPaths, path)
		}

		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		ctxSuffix := "; default context"
		if data.ConfigContext.ValueString() != "" || data.ConfigContextAuthInfo.ValueString() != "" || data.ConfigContextCluster.ValueString() != "" {
			ctxSuffix = "; overriden context"
			if data.ConfigContext.ValueString() != "" {
				overrides.CurrentContext = data.ConfigContext.ValueString()
				ctxSuffix += fmt.Sprintf("; config ctx: %s", overrides.CurrentContext)
			}
			overrides.Context = clientcmdapi.Context{}
			if data.ConfigContextAuthInfo.ValueString() != "" {
				overrides.Context.AuthInfo = data.ConfigContextAuthInfo.ValueString()
				ctxSuffix += fmt.Sprintf("; auth_info: %s", overrides.Context.AuthInfo)
			}
			if data.ConfigContextCluster.ValueString() != "" {
				overrides.Context.Cluster = data.ConfigContextCluster.ValueString()
				ctxSuffix += fmt.Sprintf("; cluster: %s", overrides.Context.Cluster)
			}
		}
	}

	// Overriding with static configuration
	overrides.ClusterInfo.InsecureSkipTLSVerify = data.Insecure.ValueBool()
	if data.ClusterCACertificate.ValueString() != "" {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(data.ClusterCACertificate.ValueString()).Bytes()
	}
	if data.ClientCertificate.ValueString() != "" {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(data.ClientCertificate.ValueString()).Bytes()
	}
	if data.Host.ValueString() != "" {
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := restclient.DefaultServerURL(data.Host.ValueString(), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse host: %s", err)
		}

		overrides.ClusterInfo.Server = host.String()
	}
	overrides.AuthInfo.Username = data.Username.ValueString()
	overrides.AuthInfo.Password = data.Password.ValueString()
	overrides.AuthInfo.Token = data.Token.ValueString()
	if data.ClientKey.ValueString() != "" {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(data.ClientKey.ValueString()).Bytes()
	}
	overrides.ClusterDefaults.ProxyURL = data.ProxyURL.ValueString()

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	// if _, err := cc.ClientConfig(); err != nil {
	// 	return nil, err
	// }
	return cc, nil
}
