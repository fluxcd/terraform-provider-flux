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
	"net/url"

	"github.com/fluxcd/flux2/pkg/manifestgen"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/gogit"
	"github.com/fluxcd/pkg/git/repository"
	runclient "github.com/fluxcd/pkg/runtime/client"
	"github.com/fluxcd/pkg/ssa"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mitchellh/go-homedir"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"

	customtypes "github.com/fluxcd/terraform-provider-flux/internal/framework/types"
	"github.com/fluxcd/terraform-provider-flux/internal/framework/validators"
	"github.com/fluxcd/terraform-provider-flux/internal/utils"
)

type providerResourceData struct {
	repositoryUrl       *url.URL
	authOpts            *git.AuthOptions
	insecureHttpAllowed bool
	ssh                 *Ssh
	http                *Http
	rcg                 *utils.RESTClientGetter
	manager             *ssa.ResourceManager
	kubeClient          client.WithWatch
}

func (prd *providerResourceData) HasClients() bool {
	return prd.rcg != nil && prd.manager != nil && prd.kubeClient != nil
}

func (prd *providerResourceData) GetGitClient(ctx context.Context, branch string, clone bool) (*gogit.Client, error) {
	tmpDir, err := manifestgen.MkdirTempAbs("", "flux-bootstrap-")
	if err != nil {
		return nil, fmt.Errorf("could not create temporary working directory for git repository: %w", err)
	}
	clientOpts := []gogit.ClientOption{gogit.WithDiskStorage()}
	if prd.insecureHttpAllowed {
		clientOpts = append(clientOpts, gogit.WithInsecureCredentialsOverHTTP())
	}
	client, err := gogit.NewClient(tmpDir, prd.authOpts, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not create git client: %w", err)
	}
	if clone {
		_, err = client.Clone(ctx, prd.repositoryUrl.String(), repository.CloneOptions{CheckoutStrategy: repository.CheckoutStrategy{Branch: branch}})
		if err != nil {
			return nil, fmt.Errorf("could not clone git repository: %w", err)
		}
	}
	return client, nil
}

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
	Url  customtypes.URL `tfsdk:"url"`
	Ssh  *Ssh            `tfsdk:"ssh"`
	Http *Http           `tfsdk:"http"`
}

type Kubernetes struct {
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

	prd := &providerResourceData{}

	// Only create Git client if configuration is set.
	if data.Git != nil {
		repositoryURL := data.Git.Url.ValueURL()
		if data.Git.Http != nil {
			repositoryURL.User = nil
			data.Git.Url = customtypes.URLValue(repositoryURL)
		}
		if data.Git.Ssh != nil {
			if data.Git.Url.ValueURL().User == nil || data.Git.Ssh.Username.ValueString() != "git" {
				repositoryURL.User = url.User(data.Git.Ssh.Username.ValueString())
				data.Git.Url = customtypes.URLValue(repositoryURL)
			}
		}
		insecureHttpAllowed := false
		if data.Git.Http != nil && data.Git.Http.InsecureHttpAllowed.ValueBool() {
			insecureHttpAllowed = true
		}
		authOpts, err := getAuthOpts(repositoryURL, data.Git.Http, data.Git.Ssh)
		if err != nil {
			resp.Diagnostics.AddError("Unable to get authentication options", err.Error())
			return
		}

		prd.repositoryUrl = repositoryURL
		prd.authOpts = authOpts
		prd.insecureHttpAllowed = insecureHttpAllowed
		prd.http = data.Git.Http
		prd.ssh = data.Git.Ssh
	}

	// Only create Kubernetes client if configuration is set.
	// If the configuration is set it is expected that it is correct and the cluster is reachable.
	if data.Kubernetes != nil {
		clientCfg, err := initializeConfiguration(ctx, data.Kubernetes)
		if err != nil {
			resp.Diagnostics.AddError("Unable to initialize configuration", err.Error())
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

		prd.rcg = rcg
		prd.manager = man
		prd.kubeClient = kubeClient
	}

	resp.ResourceData = prd
}

func (p *fluxProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSyncDataSource,
		NewInstallDataSource,
	}
}

func (p *fluxProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewBootstrapGitResource,
	}
}

func getAuthOpts(u *url.URL, h *Http, s *Ssh) (*git.AuthOptions, error) {
	switch u.Scheme {
	case "http":
		if h == nil {
			return nil, fmt.Errorf("Git URL scheme is http but http configuration is empty")
		}
		return &git.AuthOptions{
			Transport: git.HTTP,
			Username:  h.Username.ValueString(),
			Password:  h.Password.ValueString(),
		}, nil
	case "https":
		if h == nil {
			return nil, fmt.Errorf("Git URL scheme is https but http configuration is empty")
		}
		return &git.AuthOptions{
			Transport: git.HTTPS,
			Username:  h.Username.ValueString(),
			Password:  h.Password.ValueString(),
			CAFile:    []byte(h.CertificateAuthority.ValueString()),
		}, nil
	case "ssh":
		if s == nil {
			return nil, fmt.Errorf("Git URL scheme is ssh but ssh configuration is empty")
		}
		if s.PrivateKey.ValueString() != "" {
			kh, err := sourcesecret.ScanHostKey(u.Host)
			if err != nil {
				return nil, err
			}
			return &git.AuthOptions{
				Transport:  git.SSH,
				Username:   s.Username.ValueString(),
				Password:   s.Password.ValueString(),
				Identity:   []byte(s.PrivateKey.ValueString()),
				KnownHosts: kh,
			}, nil
		}
		return nil, fmt.Errorf("ssh scheme cannot be used without private key")
	default:
		return nil, fmt.Errorf("scheme %q is not supported", u.Scheme)
	}
}

func initializeConfiguration(ctx context.Context, kubernetes *Kubernetes) (clientcmd.ClientConfig, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}
	if kubernetes.ConfigPath.ValueString() != "" {
		configPaths = []string{kubernetes.ConfigPath.ValueString()}
	} else if len(kubernetes.ConfigPaths.Elements()) > 0 {
		var pp []string
		diag := kubernetes.ConfigPaths.ElementsAs(ctx, &pp, false)
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
		if kubernetes.ConfigContext.ValueString() != "" || kubernetes.ConfigContextAuthInfo.ValueString() != "" || kubernetes.ConfigContextCluster.ValueString() != "" {
			ctxSuffix = "; overriden context"
			if kubernetes.ConfigContext.ValueString() != "" {
				overrides.CurrentContext = kubernetes.ConfigContext.ValueString()
				ctxSuffix += fmt.Sprintf("; config ctx: %s", overrides.CurrentContext)
			}
			overrides.Context = clientcmdapi.Context{}
			if kubernetes.ConfigContextAuthInfo.ValueString() != "" {
				overrides.Context.AuthInfo = kubernetes.ConfigContextAuthInfo.ValueString()
				ctxSuffix += fmt.Sprintf("; auth_info: %s", overrides.Context.AuthInfo)
			}
			if kubernetes.ConfigContextCluster.ValueString() != "" {
				overrides.Context.Cluster = kubernetes.ConfigContextCluster.ValueString()
				ctxSuffix += fmt.Sprintf("; cluster: %s", overrides.Context.Cluster)
			}
		}
	}

	// Overriding with static configuration
	overrides.ClusterInfo.InsecureSkipTLSVerify = kubernetes.Insecure.ValueBool()
	if kubernetes.ClusterCACertificate.ValueString() != "" {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(kubernetes.ClusterCACertificate.ValueString()).Bytes()
	}
	if kubernetes.ClientCertificate.ValueString() != "" {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(kubernetes.ClientCertificate.ValueString()).Bytes()
	}
	if kubernetes.Host.ValueString() != "" {
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := restclient.DefaultServerURL(kubernetes.Host.ValueString(), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse host: %s", err)
		}

		overrides.ClusterInfo.Server = host.String()
	}
	overrides.AuthInfo.Username = kubernetes.Username.ValueString()
	overrides.AuthInfo.Password = kubernetes.Password.ValueString()
	overrides.AuthInfo.Token = kubernetes.Token.ValueString()
	if kubernetes.ClientKey.ValueString() != "" {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(kubernetes.ClientKey.ValueString()).Bytes()
	}
	overrides.ClusterDefaults.ProxyURL = kubernetes.ProxyURL.ValueString()

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)

	// Validate that the Kubernetes configuration is correct.
	if _, err := cc.ClientConfig(); err != nil {
		return nil, err
	}

	return cc, nil
}
