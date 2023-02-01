package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"

	"github.com/fluxcd/flux2/pkg/bootstrap"
	"github.com/fluxcd/flux2/pkg/log"
	"github.com/fluxcd/flux2/pkg/manifestgen"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/flux2/pkg/manifestgen/sync"
	"github.com/fluxcd/flux2/pkg/uninstall"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/gogit"
	"github.com/fluxcd/pkg/git/repository"
	runclient "github.com/fluxcd/pkg/runtime/client"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/terraform-provider-flux/internal/framework/planmodifiers"
	customtypes "github.com/fluxcd/terraform-provider-flux/internal/framework/types"
	"github.com/fluxcd/terraform-provider-flux/internal/framework/validators"
	"github.com/fluxcd/terraform-provider-flux/internal/utils"
)

const (
	defaultBranch      = "main"
	defaultAuthor      = "Flux"
	rfc1123LabelRegex  = `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	rfc1123LabelError  = "a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character"
	rfc1123DomainRegex = `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	rfc1123DomainError = "a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character"
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

type bootstrapGitResourceData struct {
	ID types.String `tfsdk:"id"`

	Version            types.String         `tfsdk:"version"`
	Url                customtypes.URL      `tfsdk:"url"`
	Branch             types.String         `tfsdk:"branch"`
	Path               types.String         `tfsdk:"path"`
	ClusterDomain      types.String         `tfsdk:"cluster_domain"`
	Components         types.Set            `tfsdk:"components"`
	ComponentsExtra    types.Set            `tfsdk:"components_extra"`
	ImagePullSecret    types.String         `tfsdk:"image_pull_secret"`
	LogLevel           types.String         `tfsdk:"log_level"`
	Namespace          types.String         `tfsdk:"namespace"`
	NetworkPolicy      types.Bool           `tfsdk:"network_policy"`
	Registry           customtypes.URL      `tfsdk:"registry"`
	TolerationKeys     types.Set            `tfsdk:"toleration_keys"`
	WatchAllNamespaces types.Bool           `tfsdk:"watch_all_namespaces"`
	Interval           customtypes.Duration `tfsdk:"interval"`
	SecretName         types.String         `tfsdk:"secret_name"`
	RecurseSubmodules  types.Bool           `tfsdk:"recurse_submodules"`

	AuthorName            types.String `tfsdk:"author_name"`
	AuthorEmail           types.String `tfsdk:"author_email"`
	GpgKeyRing            types.String `tfsdk:"gpg_key_ring"`
	GpgPassphrase         types.String `tfsdk:"gpg_passphrase"`
	GpgKeyID              types.String `tfsdk:"gpg_key_id"`
	CommitMessageAppendix types.String `tfsdk:"commit_message_appendix"`

	Ssh  *Ssh  `tfsdk:"ssh"`
	Http *Http `tfsdk:"http"`

	KustomizationOverride types.String `tfsdk:"kustomization_override"`

	RepositoryFiles types.Map `tfsdk:"repository_files"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &bootstrapGitResource{}
var _ resource.ResourceWithConfigure = &bootstrapGitResource{}
var _ resource.ResourceWithImportState = &bootstrapGitResource{}
var _ resource.ResourceWithModifyPlan = &bootstrapGitResource{}
var _ resource.ResourceWithValidateConfig = &bootstrapGitResource{}

type bootstrapGitResource struct {
	prd *providerResourceData
}

func NewBootstrapGitResource() resource.Resource {
	return &bootstrapGitResource{}
}

func (r *bootstrapGitResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	prd, ok := req.ProviderData.(*providerResourceData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providerResourceData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.prd = prd
}

func (r *bootstrapGitResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bootstrap_git"
}

func (r *bootstrapGitResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	defaultOpts := install.MakeDefaultOptions()
	resp.Schema = schema.Schema{
		MarkdownDescription: "Commits Flux components to a Git repository and configures a Kubernetes cluster to synchronize with the same Git repository.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"version": schema.StringAttribute{
				Description: "Flux version.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					planmodifiers.DefaultStringValue(utils.DefaultFluxVersion),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("(latest|^v.*)"), "must either be latest or start with 'v'"),
				},
			},
			"cluster_domain": schema.StringAttribute{
				Description: "The internal cluster domain.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					planmodifiers.DefaultStringValue(defaultOpts.ClusterDomain),
				},
			},
			"components": schema.SetAttribute{
				ElementType: types.StringType,
				Description: "Toolkit components to include in the install manifests.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Set{
					planmodifiers.DefaultStringSetValue(defaultOpts.Components),
				},
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(2),
					setvalidator.ValueStringsAre(stringvalidator.OneOf("source-controller", "kustomize-controller", "helm-controller", "notification-controller")),
					validators.MustContain("source-controller", "kustomize-controller"),
				},
			},
			"components_extra": schema.SetAttribute{
				ElementType: types.StringType,
				Description: "List of extra components to include in the install manifests.",
				Optional:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtMost(2),
					setvalidator.ValueStringsAre(stringvalidator.OneOf("image-reflector-controller", "image-automation-controller")),
				},
			},
			"image_pull_secret": schema.StringAttribute{
				Description: "Kubernetes secret name used for pulling the toolkit images from a private registry.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(rfc1123DomainRegex), rfc1123DomainError),
					stringvalidator.LengthAtMost(253),
				},
			},
			"log_level": schema.StringAttribute{
				Description: "Log level for toolkit components.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					planmodifiers.DefaultStringValue(defaultOpts.LogLevel),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("info", "debug", "error"),
				},
			},
			"namespace": schema.StringAttribute{
				Description: "The namespace scope for install manifests.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					planmodifiers.DefaultStringValue(defaultOpts.Namespace),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(rfc1123LabelRegex), rfc1123LabelError),
					stringvalidator.LengthAtMost(63),
				},
			},
			"network_policy": schema.BoolAttribute{
				Description: "Deny ingress access to the toolkit controllers from other namespaces using network policies.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					planmodifiers.DefaultBoolValue(defaultOpts.NetworkPolicy),
				},
			},
			"registry": schema.StringAttribute{
				CustomType:  customtypes.URLType{},
				Description: "Container registry where the toolkit images are published.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					planmodifiers.DefaultStringValue(defaultOpts.Registry),
				},
			},
			"toleration_keys": schema.SetAttribute{
				ElementType: types.StringType,
				Description: "List of toleration keys used to schedule the components pods onto nodes with matching taints.",
				Optional:    true,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(regexp.MustCompile(rfc1123LabelRegex), rfc1123LabelError),
						stringvalidator.LengthAtMost(63),
					),
				},
			},
			"watch_all_namespaces": schema.BoolAttribute{
				Description: "If true watch for custom resources in all namespaces.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					planmodifiers.DefaultBoolValue(defaultOpts.WatchAllNamespaces),
				},
			},
			"url": schema.StringAttribute{
				CustomType:  customtypes.URLType{},
				Description: "Url of git repository to bootstrap from.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.URLScheme("http", "https", "ssh"),
				},
			},
			"interval": schema.StringAttribute{
				CustomType:  customtypes.DurationType{},
				Description: "Interval at which to reconcile from bootstrap repository.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					planmodifiers.DefaultStringValue(time.Minute.String()),
				},
			},
			"path": schema.StringAttribute{
				Description: "Path relative to the repository root, when specified the cluster sync will be scoped to this path.",
				Optional:    true,
			},
			"branch": schema.StringAttribute{
				Description: "Branch in repository to reconcile from.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					planmodifiers.DefaultStringValue(defaultBranch),
				},
			},
			"recurse_submodules": schema.BoolAttribute{
				Description: "Configures the GitRepository source to initialize and include Git submodules in the artifact it produces.",
				Optional:    true,
			},
			"secret_name": schema.StringAttribute{
				Description: "Name of the secret the sync credentials can be found in or stored to.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					planmodifiers.DefaultStringValue(defaultOpts.Namespace),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(rfc1123DomainRegex), rfc1123DomainError),
					stringvalidator.LengthAtMost(253),
				},
			},
			"author_name": schema.StringAttribute{
				Description: "Author name for Git commits.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					planmodifiers.DefaultStringValue(defaultAuthor),
				},
			},
			"author_email": schema.StringAttribute{
				Description: "Author email for Git commits.",
				Optional:    true,
			},
			"gpg_key_ring": schema.StringAttribute{
				Description: "GPG key ring for signing commits.",
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
			"kustomization_override": schema.StringAttribute{
				Description: "Kustomization to override configuration set by default.",
				Optional:    true,
				Validators:  []validator.String{validators.KustomizationOverride()},
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
			"repository_files": schema.MapAttribute{
				ElementType: types.StringType,
				Description: "Git repository files created and managed by the provider.",
				Computed:    true,
			},
			"timeouts": timeouts.AttributesAll(ctx),
		},
	}
}

func (r *bootstrapGitResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data bootstrapGitResourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Url.ValueURL() != nil {
		if data.Url.ValueURL().Scheme == "ssh" && data.Http != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("http"),
				"Unexpected Attribute Configuration",
				"Did not expect http to be configured when url scheme is ssh",
			)
		}

		if (data.Url.ValueURL().Scheme == "http" || data.Url.ValueURL().Scheme == "https") && data.Ssh != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("ssh"),
				"Unexpected Attribute Configuration",
				"Did not expect ssh to be configured when url scheme is http(s)",
			)
		}

		if data.Url.ValueURL().Scheme == "http" && !data.Http.InsecureHttpAllowed.ValueBool() {
			resp.Diagnostics.AddAttributeError(
				path.Root("allow_insecure_http"),
				"Scheme Validation Error",
				"Expected allow_insecure_http to be true when url scheme is http.",
			)
		}
	}
}

func (r bootstrapGitResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip when deleting or on initial creation
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var data bootstrapGitResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Modify repository url
	if data.Http != nil {
		repositoryURL := data.Url.ValueURL()
		repositoryURL.User = nil
		data.Url = customtypes.URLValue(repositoryURL)
	}
	if data.Ssh != nil {
		if data.Url.ValueURL().User == nil || data.Ssh.Username.ValueString() != "git" {
			repositoryURL := data.Url.ValueURL()
			repositoryURL.User = url.User(data.Ssh.Username.ValueString())
			data.Url = customtypes.URLValue(repositoryURL)
		}
	}

	// Write expected repository files
	repositoryFiles, err := getExpectedRepositoryFiles(data)
	if err != nil {
		resp.Diagnostics.AddError("Getting expected repository files", err.Error())
		return
	}
	mapValue, diags := types.MapValueFrom(ctx, types.StringType, repositoryFiles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RepositoryFiles = mapValue

	diags = resp.Plan.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	// This has to be set here, probably a bug in the SDK
	diags = resp.Plan.SetAttribute(ctx, path.Root("id"), data.Namespace)
	resp.Diagnostics.Append(diags...)
}

// TODO: If kustomization file exists and not all resource files exist bootstrap will fail. This is because kustomize build is run.
func (r *bootstrapGitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.prd == nil || !r.prd.HasClients() {
		resp.Diagnostics.AddError(
			"Unconfigured Clients",
			"Expected configured provider clients.",
		)
		return
	}

	var data bootstrapGitResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := data.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	gitClient, err := getGitClient(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Git Client", err.Error())
		return
	}
	defer os.RemoveAll(gitClient.Path())

	installOpts := getInstallOptions(data)
	syncOpts := getSyncOptions(data)
	secretOpts := sourcesecret.Options{
		Name:         data.SecretName.ValueString(),
		Namespace:    data.Namespace.ValueString(),
		TargetPath:   data.Path.ValueString(),
		ManifestFile: sourcesecret.MakeDefaultOptions().ManifestFile,
	}
	if data.Http != nil {
		secretOpts.Username = data.Http.Username.ValueString()
		secretOpts.Password = data.Http.Password.ValueString()
		secretOpts.CAFile = []byte(data.Http.CertificateAuthority.ValueString())
	}
	if data.Ssh != nil {
		if data.Ssh.PrivateKey.ValueString() != "" {
			keypair, err := sourcesecret.LoadKeyPair([]byte(data.Ssh.PrivateKey.ValueString()), data.Ssh.Password.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Failed to load SSH Key Pair", err.Error())
				return
			}
			secretOpts.Keypair = keypair
			secretOpts.Password = data.Ssh.Password.ValueString()
		}
		secretOpts.SSHHostname = data.Url.ValueURL().Host
	}

	var entityList openpgp.EntityList
	if data.GpgKeyRing.ValueString() != "" {
		entityList, err = openpgp.ReadKeyRing(strings.NewReader(data.GpgKeyRing.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError("Failed to read GPG key ring", err.Error())
			return
		}
	}
	bootstrapOpts := []bootstrap.GitOption{
		bootstrap.WithRepositoryURL(data.Url.ValueURL().String()),
		bootstrap.WithBranch(data.Branch.ValueString()),
		bootstrap.WithSignature(data.AuthorName.ValueString(), data.AuthorEmail.ValueString()),
		bootstrap.WithCommitMessageAppendix(data.CommitMessageAppendix.ValueString()),
		bootstrap.WithKubeconfig(r.prd.rcg, &runclient.Options{}),
		bootstrap.WithLogger(log.NopLogger{}),
		bootstrap.WithGitCommitSigning(entityList, data.GpgPassphrase.ValueString(), data.GpgKeyID.ValueString()),
	}
	b, err := bootstrap.NewPlainGitProvider(gitClient, r.prd.kubeClient, bootstrapOpts...)
	if err != nil {
		resp.Diagnostics.AddError("Could not create bootstrap provider", err.Error())
		return
	}

	// Write own kustomization file
	if data.KustomizationOverride.ValueString() != "" {
		// Need to write empty gotk-components and gotk-sync because other wise Kustomize will not work.
		basePath := filepath.Join(gitClient.Path(), data.Path.ValueString(), data.Namespace.ValueString())
		files := map[string]io.Reader{
			filepath.Join(basePath, konfig.DefaultKustomizationFileName()): strings.NewReader(data.KustomizationOverride.ValueString()),
			filepath.Join(basePath, "gotk-components.yaml"):                &strings.Reader{},
			filepath.Join(basePath, "gotk-sync.yaml"):                      &strings.Reader{},
		}
		commit, signer, err := getCommit(data, "Add kustomize override")
		if err != nil {
			resp.Diagnostics.AddError("Unable to create commit", err.Error())
			return
		}
		_, err = gitClient.Commit(commit, signer, repository.WithFiles(files))
		if err != nil {
			resp.Diagnostics.AddError("Could not create bootstrap provider", err.Error())
			return
		}
	}

	manifestsBase := ""
	err = bootstrap.Run(ctx, b, manifestsBase, installOpts, secretOpts, syncOpts, 2*time.Second, createTimeout)
	if err != nil {
		resp.Diagnostics.AddError("Bootstrap run error", err.Error())
		return
	}

	// TODO: Figure out a better way to track files commited to git
	repositoryFiles := map[string]string{}
	files := []string{install.MakeDefaultOptions().ManifestFile, sync.MakeDefaultOptions().ManifestFile, konfig.DefaultKustomizationFileName()}
	for _, f := range files {
		path := filepath.Join(data.Path.ValueString(), data.Namespace.ValueString(), f)
		b, err := os.ReadFile(filepath.Join(gitClient.Path(), path))
		if err != nil {
			resp.Diagnostics.AddError("Could not read repository state", err.Error())
			return
		}
		repositoryFiles[path] = string(b)
	}
	mapValue, diags := types.MapValueFrom(ctx, types.StringType, repositoryFiles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RepositoryFiles = mapValue

	data.ID = data.Namespace
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// TODO: Consider if more value reading should be done here to detect drift. Similar to how import works.
func (r *bootstrapGitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data bootstrapGitResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := data.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	gitClient, err := getGitClient(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Git Client", err.Error())
		return
	}
	defer os.RemoveAll(gitClient.Path())

	// Sync git repository with Terraform state.
	repositoryFiles := map[string]string{}
	for k := range data.RepositoryFiles.Elements() {
		path := filepath.Join(gitClient.Path(), k)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			tflog.Debug(ctx, "Skip reading file that no longer exists in git repository", map[string]interface{}{"path": path})
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read file in git repository", err.Error())
			return
		}
		repositoryFiles[k] = string(b)
	}
	mapValue, diags := types.MapValueFrom(ctx, types.StringType, repositoryFiles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RepositoryFiles = mapValue

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// TODO: Verify Flux components after updating Git
func (r bootstrapGitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data bootstrapGitResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := data.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	gitClient, err := getGitClient(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Git Client", err.Error())
		return
	}
	defer os.RemoveAll(gitClient.Path())

	// Files should be removed if they are present in the state but not the plan.
	previousRepositoryFiles := types.MapNull(types.StringType)
	diags = req.State.GetAttribute(ctx, path.Root("repository_files"), &previousRepositoryFiles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	for k := range previousRepositoryFiles.Elements() {
		_, ok := data.RepositoryFiles.Elements()[k]
		if ok {
			continue
		}
		path := filepath.Join(gitClient.Path(), k)
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			tflog.Debug(ctx, "Skipping removing no longer tracked file as it does not exist", map[string]interface{}{"path": path})
			continue
		}
		if err != nil {
			resp.Diagnostics.AddError("Could not stat no longer tracked file", err.Error())
			return
		}
		err = os.Remove(path)
		if err != nil {
			resp.Diagnostics.AddError("Could not remove no longer tracked file", err.Error())
			return
		}
	}

	// Write expected file contents to repo
	repositoryFiles := map[string]string{}
	diags = data.RepositoryFiles.ElementsAs(ctx, &repositoryFiles, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	files := map[string]io.Reader{}
	for k, v := range repositoryFiles {
		files[k] = strings.NewReader(v)
	}
	commit, signer, err := getCommit(data, "Update Flux")
	if err != nil {
		resp.Diagnostics.AddError("Unable to create commit", err.Error())
		return
	}
	_, err = gitClient.Commit(commit, signer, repository.WithFiles(files))
	if err != nil && !errors.Is(err, git.ErrNoStagedFiles) {
		resp.Diagnostics.AddError("Unable to commit updated files", err.Error())
		return
	}
	// Only push if changes are committed
	if err == nil {
		err = gitClient.Push(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Unable to push updated files", err.Error())
			return
		}
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r bootstrapGitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.prd == nil || !r.prd.HasClients() {
		resp.Diagnostics.AddError(
			"Unconfigured Clients",
			"Expected configured provider clients.",
		)
		return
	}

	var data bootstrapGitResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := data.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	gitClient, err := getGitClient(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Git Client", err.Error())
		return
	}
	defer os.RemoveAll(gitClient.Path())

	// TODO: Uninstall fails when flux-system namespace does not exist
	err = uninstall.Components(ctx, log.NopLogger{}, r.prd.kubeClient, data.Namespace.ValueString(), false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove Flux components", err.Error())
		return
	}
	err = uninstall.Finalizers(ctx, log.NopLogger{}, r.prd.kubeClient, false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove finalizers", err.Error())
		return
	}
	err = uninstall.CustomResourceDefinitions(ctx, log.NopLogger{}, r.prd.kubeClient, false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove CRDs", err.Error())
		return
	}
	err = uninstall.Namespace(ctx, log.NopLogger{}, r.prd.kubeClient, data.Namespace.ValueString(), false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove namespace", err.Error())
		return
	}

	// Remove all tracked files from git
	for k := range data.RepositoryFiles.Elements() {
		path := filepath.Join(gitClient.Path(), k)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			tflog.Debug(ctx, "Skipping file removal as the file does not exist", map[string]interface{}{"path": path})
			continue
		}
		err := os.Remove(path)
		if err != nil {
			resp.Diagnostics.AddError("Could not remove file from git repository", err.Error())
			return
		}
	}
	commit, signer, err := getCommit(data, "Uninstall Flux")
	if err != nil {
		resp.Diagnostics.AddError("Unable to create commit", err.Error())
		return
	}
	_, err = gitClient.Commit(commit, signer)
	if err != nil {
		resp.Diagnostics.AddError("Unable to commit removed file(s)", err.Error())
		return
	}
	err = gitClient.Push(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to push removed file(s)", err.Error())
		return
	}
}

// TODO: Validate Flux installation before proceeding with import
func (r *bootstrapGitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if r.prd == nil || !r.prd.HasClients() {
		resp.Diagnostics.AddError(
			"Unconfigured Clients",
			"Expected configured provider clients.",
		)
		return
	}

	data := bootstrapGitResourceData{}
	data.ID = types.StringValue(req.ID)
	data.Namespace = data.ID

	// It is impossible to determine the Author so we just set it to the default
	data.AuthorName = types.StringValue(defaultAuthor)

	// Set values that cant be null
	data.TolerationKeys = types.SetNull(types.StringType)

	// Get Network NetworkPolicy
	networkPolicy := networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-webhooks",
			Namespace: data.Namespace.ValueString(),
		},
	}
	err := r.prd.kubeClient.Get(ctx, client.ObjectKeyFromObject(&networkPolicy), &networkPolicy)
	if err != nil && !k8serrors.IsNotFound(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not get NetworkPolicy %s/%s", networkPolicy.Namespace, networkPolicy.Name), err.Error())
		return
	}
	data.NetworkPolicy = types.BoolValue(true)
	if err != nil && k8serrors.IsNotFound(err) {
		data.NetworkPolicy = types.BoolValue(false)
	}

	// Get values from kustomize-controller Deployment
	kustomizeDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kustomize-controller",
			Namespace: data.Namespace.ValueString(),
		},
	}
	if err := r.prd.kubeClient.Get(ctx, client.ObjectKeyFromObject(&kustomizeDeployment), &kustomizeDeployment); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not get Deployment %s/%s", kustomizeDeployment.Namespace, kustomizeDeployment.Name), err.Error())
		return
	}
	managerContainer, err := getContainer(kustomizeDeployment.Spec.Template.Spec.Containers, "manager")
	if err != nil {
		resp.Diagnostics.AddError("Could not get manager container", err.Error())
		return
	}

	// Get Flux Version beeing used
	version, ok := kustomizeDeployment.Labels["app.kubernetes.io/version"]
	if !ok {
		resp.Diagnostics.AddError("Version label not found", "Label is not present in kustomize-controller Deployment")
		return
	}
	data.Version = types.StringValue(version)

	// Get Image Registry
	ref, err := name.ParseReference(managerContainer.Image)
	if err != nil {
		resp.Diagnostics.AddError("Could not parse image reference", err.Error())
		return
	}
	regstryStr := fmt.Sprintf("%s/%s", ref.Context().RegistryStr(), strings.Split(ref.Context().RepositoryStr(), "/")[0])
	u, err := url.Parse(regstryStr)
	if err != nil {
		resp.Diagnostics.AddError("Could not parse url", err.Error())
		return
	}
	data.Registry = customtypes.URLValue(u)

	// Get image pull secrets
	data.ImagePullSecret = types.StringNull()
	if len(kustomizeDeployment.Spec.Template.Spec.ImagePullSecrets) > 0 {
		data.ImagePullSecret = types.StringValue(kustomizeDeployment.Spec.Template.Spec.ImagePullSecrets[0].Name)
	}

	// Get if watching all namespace
	value, err := getArgValue(managerContainer.Args, "--watch-all-namespaces")
	if err != nil {
		resp.Diagnostics.AddError("Could not get arg", err.Error())
		return
	}
	watchAllNamespaces, err := strconv.ParseBool(value)
	if err != nil {
		resp.Diagnostics.AddError("Could not convert watch all namespaces from string to bool", err.Error())
		return
	}
	data.WatchAllNamespaces = types.BoolValue(watchAllNamespaces)

	// Get log level
	value, err = getArgValue(managerContainer.Args, "--log-level")
	if err != nil {
		resp.Diagnostics.AddError("Could not get arg", err.Error())
		return
	}
	data.LogLevel = types.StringValue(value)

	// Get cluster domain
	value, err = getArgValue(managerContainer.Args, "--events-addr")
	if err != nil {
		resp.Diagnostics.AddError("Could not get arg", err.Error())
		return
	}
	eventsUrl, err := url.Parse(value)
	if err != nil {
		resp.Diagnostics.AddError("Could not parse events address", err.Error())
		return
	}
	// TODO: Probably smarter to remove what we know comes before the cluster domain and remove that
	host := strings.TrimSuffix(eventsUrl.Host, ".")
	c := strings.Split(host, ".")
	clusterDomain := strings.Join(c[len(c)-2:], ".")
	data.ClusterDomain = types.StringValue(clusterDomain)

	// Get values from flux-system GitRepository
	gitRepository := sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Namespace.ValueString(),
			Namespace: data.Namespace.ValueString(),
		},
	}
	if err := r.prd.kubeClient.Get(ctx, client.ObjectKeyFromObject(&gitRepository), &gitRepository); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not get GitRepository %s/%s", gitRepository.Namespace, gitRepository.Name), err.Error())
		return
	}
	repositoryUrl, err := url.Parse(gitRepository.Spec.URL)
	if err != nil {
		resp.Diagnostics.AddError("Unable to parse repository url", err.Error())
		return
	}
	data.Url = customtypes.URLValue(repositoryUrl)
	data.Branch = types.StringValue(gitRepository.Spec.Reference.Branch)
	data.SecretName = types.StringValue(gitRepository.Spec.SecretRef.Name)
	data.Interval = customtypes.DurationValue(gitRepository.Spec.Interval.Duration)

	// Get values from flux-system Kustomization
	kustomization := kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Namespace.ValueString(),
			Namespace: data.Namespace.ValueString(),
		},
	}
	if err := r.prd.kubeClient.Get(ctx, client.ObjectKeyFromObject(&kustomization), &kustomization); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not get Kustomization %s/%s", kustomization.Namespace, kustomization.Name), err.Error())
		return
	}
	// Only set path value if path is something other than nil. This is to be consistent with the default value.
	data.Path = types.StringNull()
	path := strings.TrimPrefix(kustomization.Spec.Path, "./")
	if path != "" {
		data.Path = types.StringValue(path)
	}

	// Get git credentials
	repositorySecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.SecretName.ValueString(),
			Namespace: data.Namespace.ValueString(),
		},
	}
	if err := r.prd.kubeClient.Get(ctx, client.ObjectKeyFromObject(&repositorySecret), &repositorySecret); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not get Secret %s/%s", repositorySecret.Namespace, repositorySecret.Name), err.Error())
		return
	}
	switch data.Url.ValueURL().Scheme {
	case "http":
		data.Http = &Http{
			Username:            types.StringValue(string(repositorySecret.Data[sourcesecret.UsernameSecretKey])),
			Password:            types.StringValue(string(repositorySecret.Data[sourcesecret.PasswordSecretKey])),
			InsecureHttpAllowed: types.BoolValue(true),
		}
	case "https":
		data.Http = &Http{
			Username:             types.StringValue(string(repositorySecret.Data[sourcesecret.UsernameSecretKey])),
			Password:             types.StringValue(string(repositorySecret.Data[sourcesecret.PasswordSecretKey])),
			CertificateAuthority: types.StringValue(string(repositorySecret.Data[sourcesecret.CAFileSecretKey])),
		}
	case "ssh":
		username := "git"
		if data.Url.ValueURL().User.Username() != "" {
			username = data.Url.ValueURL().User.Username()
		}
		data.Ssh = &Ssh{
			Username:   types.StringValue(username),
			PrivateKey: types.StringValue(string(repositorySecret.Data[sourcesecret.PrivateKeySecretKey])),
		}
	}

	// Check which components are present and which are not
	components := []attr.Value{}
	for _, c := range install.MakeDefaultOptions().Components {
		dep := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c,
				Namespace: data.Namespace.ValueString(),
			},
		}
		err := r.prd.kubeClient.Get(ctx, client.ObjectKeyFromObject(&dep), &dep)
		if err != nil && k8serrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Could not get Deployment %s/%s", dep.Namespace, dep.Name), err.Error())
			return
		}
		components = append(components, types.StringValue(c))
	}
	componentsSet, diags := types.SetValue(types.StringType, components)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Components = componentsSet

	componentsExtra := []attr.Value{}
	for _, c := range install.MakeDefaultOptions().ComponentsExtra {
		dep := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c,
				Namespace: data.Namespace.ValueString(),
			},
		}
		err := r.prd.kubeClient.Get(ctx, client.ObjectKeyFromObject(&dep), &dep)
		if err != nil && k8serrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Could not get Deployment %s/%s", dep.Namespace, dep.Name), err.Error())
			return
		}
		componentsExtra = append(components, types.StringValue(c))
	}
	componentsExtraSet, diags := types.SetValue(types.StringType, componentsExtra)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ComponentsExtra = types.SetNull(types.StringType)
	if len(componentsExtra) > 0 {
		data.ComponentsExtra = componentsExtraSet
	}

	// Set expected repository files
	repositoryFiles, err := getExpectedRepositoryFiles(data)
	if err != nil {
		resp.Diagnostics.AddError("Getting expected repository files", err.Error())
		return
	}
	mapValue, diags := types.MapValueFrom(ctx, types.StringType, repositoryFiles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RepositoryFiles = mapValue

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func getKustomizationFile(data bootstrapGitResourceData) string {
	if data.KustomizationOverride.ValueString() != "" {
		return data.KustomizationOverride.ValueString()
	}
	return `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- gotk-components.yaml
- gotk-sync.yaml
`
}

func getGitClient(ctx context.Context, data bootstrapGitResourceData) (*gogit.Client, error) {
	authOpts, err := getAuthOpts(data.Url.ValueURL(), data.Http, data.Ssh)
	if err != nil {
		return nil, err
	}
	tmpDir, err := manifestgen.MkdirTempAbs("", "flux-bootstrap-")
	if err != nil {
		return nil, fmt.Errorf("could not create temporary working directory for git repository: %w", err)
	}
	clientOpts := []gogit.ClientOption{gogit.WithDiskStorage()}
	if data.Http != nil && data.Http.InsecureHttpAllowed.ValueBool() {
		clientOpts = append(clientOpts, gogit.WithInsecureCredentialsOverHTTP())
	}
	client, err := gogit.NewClient(tmpDir, authOpts, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not create git client: %w", err)
	}
	_, err = client.Clone(ctx, data.Url.ValueURL().String(), repository.CloneOptions{CheckoutStrategy: repository.CheckoutStrategy{Branch: data.Branch.ValueString()}})
	if err != nil {
		return nil, fmt.Errorf("could not clone git repository: %w", err)
	}
	return client, nil
}

func getAuthOpts(u *url.URL, h *Http, s *Ssh) (*git.AuthOptions, error) {
	switch u.Scheme {
	case "http":
		return &git.AuthOptions{
			Transport: git.HTTP,
			Username:  h.Username.ValueString(),
			Password:  h.Password.ValueString(),
		}, nil
	case "https":
		return &git.AuthOptions{
			Transport: git.HTTPS,
			Username:  h.Username.ValueString(),
			Password:  h.Password.ValueString(),
			CAFile:    []byte(h.CertificateAuthority.ValueString()),
		}, nil
	case "ssh":
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

func getCommit(data bootstrapGitResourceData, message string) (git.Commit, repository.CommitOption, error) {
	var entityList openpgp.EntityList
	if data.GpgKeyRing.ValueString() != "" {
		var err error
		entityList, err = openpgp.ReadKeyRing(strings.NewReader(data.GpgKeyRing.ValueString()))
		if err != nil {
			return git.Commit{}, nil, fmt.Errorf("Failed to read GPG key ring: %w", err)
		}
	}
	var signer *openpgp.Entity
	if entityList != nil {
		var err error
		signer, err = getOpenPgpEntity(entityList, data.GpgPassphrase.ValueString(), data.GpgKeyID.ValueString())
		if err != nil {
			return git.Commit{}, nil, fmt.Errorf("failed to generate OpenPGP entity: %w", err)
		}
	}
	if data.CommitMessageAppendix.ValueString() != "" {
		message = message + "\n\n" + data.CommitMessageAppendix.ValueString()
	}
	commit := git.Commit{
		Author: git.Signature{
			Name:  data.AuthorName.ValueString(),
			Email: data.AuthorEmail.ValueString(),
		},
		Message: message,
	}
	return commit, repository.WithSigner(signer), nil
}

func getOpenPgpEntity(keyRing openpgp.EntityList, passphrase, keyID string) (*openpgp.Entity, error) {
	if len(keyRing) == 0 {
		return nil, fmt.Errorf("empty GPG key ring")
	}

	var entity *openpgp.Entity
	if keyID != "" {
		if strings.HasPrefix(keyID, "0x") {
			keyID = strings.TrimPrefix(keyID, "0x")
		}
		if len(keyID) != 16 {
			return nil, fmt.Errorf("invalid GPG key id length; expected %d, got %d", 16, len(keyID))
		}
		keyID = strings.ToUpper(keyID)

		for _, ent := range keyRing {
			if ent.PrimaryKey.KeyIdString() == keyID {
				entity = ent
			}
		}

		if entity == nil {
			return nil, fmt.Errorf("no GPG keyring matching key id '%s' found", keyID)
		}
		if entity.PrivateKey == nil {
			return nil, fmt.Errorf("keyring does not contain private key for key id '%s'", keyID)
		}
	} else {
		entity = keyRing[0]
	}

	err := entity.PrivateKey.Decrypt([]byte(passphrase))
	if err != nil {
		return nil, fmt.Errorf("unable to decrypt GPG private key: %w", err)
	}

	return entity, nil
}

func getInstallOptions(data bootstrapGitResourceData) install.Options {
	components := []string{}
	data.Components.ElementsAs(context.Background(), &components, false)
	sort.Strings(components)
	componentsExtra := []string{}
	data.ComponentsExtra.ElementsAs(context.Background(), &componentsExtra, false)
	sort.Strings(componentsExtra)
	components = append(components, componentsExtra...)

	tolerationKeys := []string{}
	data.TolerationKeys.ElementsAs(context.Background(), &tolerationKeys, false)
	sort.Strings(tolerationKeys)

	installOptions := install.Options{
		BaseURL:                install.MakeDefaultOptions().BaseURL,
		Version:                data.Version.ValueString(),
		Namespace:              data.Namespace.ValueString(),
		Components:             components,
		Registry:               data.Registry.ValueURL().String(),
		ImagePullSecret:        data.ImagePullSecret.ValueString(),
		WatchAllNamespaces:     data.WatchAllNamespaces.ValueBool(),
		NetworkPolicy:          data.NetworkPolicy.ValueBool(),
		LogLevel:               data.LogLevel.ValueString(),
		NotificationController: install.MakeDefaultOptions().NotificationController,
		ManifestFile:           install.MakeDefaultOptions().ManifestFile,
		Timeout:                install.MakeDefaultOptions().Timeout,
		TargetPath:             data.Path.ValueString(),
		ClusterDomain:          data.ClusterDomain.ValueString(),
		TolerationKeys:         tolerationKeys,
	}
	return installOptions
}

func getSyncOptions(data bootstrapGitResourceData) sync.Options {
	syncOpts := sync.Options{
		Interval:          data.Interval.ValueDuration(),
		Name:              data.Namespace.ValueString(),
		Namespace:         data.Namespace.ValueString(),
		URL:               data.Url.ValueURL().String(),
		Branch:            data.Branch.ValueString(),
		Secret:            data.SecretName.ValueString(),
		TargetPath:        data.Path.ValueString(),
		ManifestFile:      sync.MakeDefaultOptions().ManifestFile,
		RecurseSubmodules: data.RecurseSubmodules.ValueBool(),
	}
	return syncOpts
}

func getExpectedRepositoryFiles(data bootstrapGitResourceData) (map[string]string, error) {
	repositoryFiles := map[string]string{}
	installOpts := getInstallOptions(data)
	installManifests, err := install.Generate(installOpts, "")
	if err != nil {
		return nil, fmt.Errorf("Could not generate install manifests: %w", err)
	}
	repositoryFiles[installManifests.Path] = installManifests.Content
	syncOpts := getSyncOptions(data)
	syncManifests, err := sync.Generate(syncOpts)
	if err != nil {
		return nil, fmt.Errorf("Could not generate sync manifests: %w", err)
	}
	repositoryFiles[syncManifests.Path] = syncManifests.Content
	repositoryFiles[filepath.Join(data.Path.ValueString(), data.Namespace.ValueString(), konfig.DefaultKustomizationFileName())] = getKustomizationFile(data)
	return repositoryFiles, nil
}

func getContainer(containers []corev1.Container, name string) (corev1.Container, error) {
	for _, c := range containers {
		return c, nil
	}
	return corev1.Container{}, fmt.Errorf("could not find container: %s", name)
}

func getArgValue(args []string, name string) (string, error) {
	for _, arg := range args {
		if strings.HasPrefix(arg, name) {
			_, after, ok := strings.Cut(arg, "=")
			if !ok {
				return "", fmt.Errorf("could not split arg: %s", arg)
			}
			return after, nil
		}
	}
	return "", fmt.Errorf("arg with name not found: %s", name)
}
