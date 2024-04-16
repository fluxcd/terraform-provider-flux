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

	"github.com/fluxcd/flux2/v2/pkg/manifestgen"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/conditions"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/konfig"

	"github.com/fluxcd/flux2/v2/pkg/bootstrap"
	"github.com/fluxcd/flux2/v2/pkg/log"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sourcesecret"
	"github.com/fluxcd/flux2/v2/pkg/manifestgen/sync"
	"github.com/fluxcd/flux2/v2/pkg/uninstall"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/git/repository"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	customtypes "github.com/fluxcd/terraform-provider-flux/internal/framework/types"
	"github.com/fluxcd/terraform-provider-flux/internal/framework/validators"
	"github.com/fluxcd/terraform-provider-flux/internal/utils"
)

const (
	defaultCreateTimeout = 15 * time.Minute
	defaultReadTimeout   = 5 * time.Minute
	defaultUpdateTimeout = 15 * time.Minute
	defaultDeleteTimeout = 5 * time.Minute

	rfc1123LabelRegex  = `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	rfc1123LabelError  = "a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-', and must start and end with an alphanumeric character"
	rfc1123DomainRegex = `^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	rfc1123DomainError = "a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character"
	tolerationKeyRegex = `^[A-Za-z0-9]([A-Za-z0-9._-]*)$`
	tolerationKeyError = "a toleration key must begin with a letter or number, and may contain letters, numbers, hyphens, dots, and underscores."

	missingConfiguration                   = "Missing configuration"
	bootstrapGitResourceMissingConfigError = "Git and Kubernetes configuration not found"
)

type bootstrapGitResourceData struct {
	ClusterDomain         types.String         `tfsdk:"cluster_domain"`
	Components            types.Set            `tfsdk:"components"`
	ComponentsExtra       types.Set            `tfsdk:"components_extra"`
	DeleteGitManifests    types.Bool           `tfsdk:"delete_git_manifests"`
	DisableSecretCreation types.Bool           `tfsdk:"disable_secret_creation"`
	EmbeddedManifests     types.Bool           `tfsdk:"embedded_manifests"`
	ID                    types.String         `tfsdk:"id"`
	ImagePullSecret       types.String         `tfsdk:"image_pull_secret"`
	Interval              customtypes.Duration `tfsdk:"interval"`
	KeepNamespace         types.Bool           `tfsdk:"keep_namespace"`
	KustomizationOverride types.String         `tfsdk:"kustomization_override"`
	LogLevel              types.String         `tfsdk:"log_level"`
	ManifestsPath         types.String         `tfsdk:"manifests_path"`
	Namespace             types.String         `tfsdk:"namespace"`
	NetworkPolicy         types.Bool           `tfsdk:"network_policy"`
	Path                  types.String         `tfsdk:"path"`
	RecurseSubmodules     types.Bool           `tfsdk:"recurse_submodules"`
	Registry              customtypes.URL      `tfsdk:"registry"`
	RepositoryFiles       types.Map            `tfsdk:"repository_files"`
	SecretName            types.String         `tfsdk:"secret_name"`
	Timeouts              timeouts.Value       `tfsdk:"timeouts"`
	TolerationKeys        types.Set            `tfsdk:"toleration_keys"`
	Version               types.String         `tfsdk:"version"`
	WatchAllNamespaces    types.Bool           `tfsdk:"watch_all_namespaces"`
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &bootstrapGitResource{}
var _ resource.ResourceWithConfigure = &bootstrapGitResource{}
var _ resource.ResourceWithImportState = &bootstrapGitResource{}
var _ resource.ResourceWithModifyPlan = &bootstrapGitResource{}

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
	componentsSet, diags := types.SetValueFrom(ctx, types.StringType, defaultOpts.Components)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Commits Flux components to a Git repository and configures a Kubernetes cluster to synchronize with the same Git repository.",
		Attributes: map[string]schema.Attribute{
			"cluster_domain": schema.StringAttribute{
				Description: fmt.Sprintf("The internal cluster domain. Defaults to `%s`", defaultOpts.ClusterDomain),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultOpts.ClusterDomain),
			},
			"components": schema.SetAttribute{
				ElementType: types.StringType,
				Description: fmt.Sprintf("Toolkit components to include in the install manifests. Defaults to `%s`", defaultOpts.Components),
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(componentsSet),
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
			"delete_git_manifests": schema.BoolAttribute{
				Description: "Delete manifests from git repository. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"disable_secret_creation": schema.BoolAttribute{
				Description: "Use the existing secret for flux controller and don't create one from bootstrap",
				Optional:    true,
			},
			"embedded_manifests": schema.BoolAttribute{
				Description: "When enabled, the Flux manifests will be extracted from the provider binary instead of being downloaded from GitHub.com. Defaults to `false`.",
				Optional:    true,
			},
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			"interval": schema.StringAttribute{
				CustomType:  customtypes.DurationType{},
				Description: fmt.Sprintf("Interval at which to reconcile from bootstrap repository. Defaults to `%s`.", time.Minute.String()),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(time.Minute.String()),
			},
			"keep_namespace": schema.BoolAttribute{
				Description: "Keep the namespace after uninstalling Flux components. Defaults to `false`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"kustomization_override": schema.StringAttribute{
				Description: "Kustomization to override configuration set by default.",
				Optional:    true,
				Validators:  []validator.String{validators.KustomizationOverride()},
			},
			"log_level": schema.StringAttribute{
				Description: fmt.Sprintf("Log level for toolkit components. Defaults to `%s`.", defaultOpts.LogLevel),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultOpts.LogLevel),
				Validators: []validator.String{
					stringvalidator.OneOf("info", "debug", "error"),
				},
			},
			"manifests_path": schema.StringAttribute{
				Description:        fmt.Sprintf("The install manifests are built from a GitHub release or kustomize overlay if using a local path. Defaults to `%s`.", defaultOpts.BaseURL),
				Optional:           true,
				DeprecationMessage: "This attribute is deprecated. Use the `embedded_manifests` attribute when running bootstrap on air-gapped environments.",
			},
			"namespace": schema.StringAttribute{
				Description: fmt.Sprintf("The namespace scope for install manifests. Defaults to `%s`. It will be created if it does not exist.", defaultOpts.Namespace),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultOpts.Namespace),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(rfc1123LabelRegex), rfc1123LabelError),
					stringvalidator.LengthAtMost(63),
				},
			},
			"network_policy": schema.BoolAttribute{
				Description: fmt.Sprintf("Deny ingress access to the toolkit controllers from other namespaces using network policies. Defaults to `%v`.", defaultOpts.NetworkPolicy),
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultOpts.NetworkPolicy),
			},
			"path": schema.StringAttribute{
				Description: "Path relative to the repository root, when specified the cluster sync will be scoped to this path (immutable).",
				Optional:    true,
			},
			"recurse_submodules": schema.BoolAttribute{
				Description: "Configures the GitRepository source to initialize and include Git submodules in the artifact it produces.",
				Optional:    true,
			},
			"registry": schema.StringAttribute{
				CustomType:  customtypes.URLType{},
				Description: fmt.Sprintf("Container registry where the toolkit images are published. Defaults to `%s`.", defaultOpts.Registry),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultOpts.Registry),
			},
			"repository_files": schema.MapAttribute{
				ElementType: types.StringType,
				Description: "Git repository files created and managed by the provider.",
				Computed:    true,
			},
			"secret_name": schema.StringAttribute{
				Description: fmt.Sprintf("Name of the secret the sync credentials can be found in or stored to. Defaults to `%s`.", defaultOpts.Namespace),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultOpts.Namespace),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(rfc1123DomainRegex), rfc1123DomainError),
					stringvalidator.LengthAtMost(253),
				},
			},
			"timeouts": timeouts.AttributesAll(ctx),
			"toleration_keys": schema.SetAttribute{
				ElementType: types.StringType,
				Description: "List of toleration keys used to schedule the components pods onto nodes with matching taints.",
				Optional:    true,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(regexp.MustCompile(tolerationKeyRegex), tolerationKeyError),
						stringvalidator.LengthAtMost(253),
					),
				},
			},
			"version": schema.StringAttribute{
				Description: fmt.Sprintf("Flux version. Defaults to `%s`. Has no effect when `embedded_manifests` is enabled.", utils.DefaultFluxVersion),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(utils.DefaultFluxVersion),
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("(latest|^v.*)"), "must either be latest or start with 'v'"),
				},
			},
			"watch_all_namespaces": schema.BoolAttribute{
				Description: fmt.Sprintf("If true watch for custom resources in all namespaces. Defaults to `%v`.", defaultOpts.WatchAllNamespaces),
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultOpts.WatchAllNamespaces),
			},
		},
	}
}

// ModifyPlan sets the desired Git repository files to be managed by the provider.
func (r bootstrapGitResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.prd == nil {
		resp.Diagnostics.AddError(missingConfiguration, bootstrapGitResourceMissingConfigError)
	}

	// Skip when deleting or on initial creation.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var data bootstrapGitResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write expected repository files.
	repositoryFiles, err := getExpectedRepositoryFiles(data, r.prd.GetRepositoryURL(), r.prd.git.Branch.ValueString())
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
}

// Create pushes the Flux manifests in the Git repository, installs the Flux controllers on the cluster
// and configures Flux to sync the cluster state with the given Git repository path.
func (r *bootstrapGitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.prd == nil {
		resp.Diagnostics.AddError(missingConfiguration, bootstrapGitResourceMissingConfigError)
	}

	var data bootstrapGitResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := data.Timeouts.Create(ctx, defaultCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	kubeClient, err := r.prd.GetKubernetesClient()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes Client", err.Error())
		return
	}

	gitClient, err := r.prd.CloneRepository(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Git Client", err.Error())
		return
	}
	defer os.RemoveAll(gitClient.Path())

	installOpts := getInstallOptions(data)
	syncOpts := getSyncOptions(data, r.prd.GetRepositoryURL(), r.prd.git.Branch.ValueString())
	var secretOpts sourcesecret.Options
	if data.DisableSecretCreation.ValueBool() {
		secretOpts = sourcesecret.Options{
			Name:      data.SecretName.ValueString(),
			Namespace: data.Namespace.ValueString(),
		}
	} else {
		secretOpts, err = r.prd.GetSecretOptions(data.SecretName.ValueString(), data.Namespace.ValueString(), data.Path.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Could not get secret options", err.Error())
			return
		}
	}

	bootstrapOpts, err := r.prd.GetBootstrapOptions()
	if err != nil {
		resp.Diagnostics.AddError("Could not get bootstrap options", err.Error())
		return
	}
	bootstrapProvider, err := bootstrap.NewPlainGitProvider(gitClient, kubeClient, bootstrapOpts...)
	if err != nil {
		resp.Diagnostics.AddError("Could not create bootstrap provider", err.Error())
		return
	}

	// Write own kustomization file
	if data.KustomizationOverride.ValueString() != "" {
		// Need to write empty gotk-components and gotk-sync because otherwise Kustomize will not work.
		basePath := filepath.Join(gitClient.Path(), data.Path.ValueString(), data.Namespace.ValueString())
		files := map[string]io.Reader{
			filepath.Join(basePath, konfig.DefaultKustomizationFileName()): strings.NewReader(data.KustomizationOverride.ValueString()),
			filepath.Join(basePath, installOpts.ManifestFile):              &strings.Reader{},
			filepath.Join(basePath, syncOpts.ManifestFile):                 &strings.Reader{},
		}
		commit, signer, err := r.prd.CreateCommit("Init Flux with kustomize override")
		if err != nil {
			resp.Diagnostics.AddError("Unable to create kustomize override commit", err.Error())
			return
		}
		_, err = gitClient.Commit(commit, signer, repository.WithFiles(files))
		if err != nil {
			resp.Diagnostics.AddError("Unable to commit kustomize override", err.Error())
			return
		}
	}

	manifestsBase := ""
	if data.EmbeddedManifests.ValueBool() {
		manifestsBase = EmbeddedManifests
	}
	err = bootstrap.Run(ctx, bootstrapProvider, manifestsBase, installOpts, secretOpts, syncOpts, 2*time.Second, timeout)
	if err != nil {
		resp.Diagnostics.AddError("Bootstrap run error", err.Error())
		return
	}

	repositoryFiles := map[string]string{}
	files := []string{installOpts.ManifestFile, syncOpts.ManifestFile, konfig.DefaultKustomizationFileName()}
	for _, f := range files {
		filePath := filepath.Join(data.Path.ValueString(), data.Namespace.ValueString(), f)
		b, err := os.ReadFile(filepath.Join(gitClient.Path(), filePath))
		if err != nil {
			resp.Diagnostics.AddError("Could not read repository state", err.Error())
			return
		}
		repositoryFiles[filePath] = string(b)
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

// Read pulls the Flux manifests from the Git repository to detect drift
// and checks the health of the Flux controllers in the cluster.
// If the Flux controllers are not healthy, the state is marked as needing an update.
// TODO: Handle Git auth key rotation.
func (r *bootstrapGitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.prd == nil {
		resp.Diagnostics.AddError(missingConfiguration, bootstrapGitResourceMissingConfigError)
	}

	var data bootstrapGitResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := data.Timeouts.Read(ctx, defaultCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	gitClient, err := r.prd.CloneRepository(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Git Client", err.Error())
		return
	}
	defer os.RemoveAll(gitClient.Path())

	// Detect drift for the Flux manifests stored in Git.
	repositoryFiles := map[string]string{}
	for k := range data.RepositoryFiles.Elements() {
		filePath := filepath.Join(gitClient.Path(), k)
		if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
			tflog.Debug(ctx, "Skip reading file that no longer exists in git repository", map[string]interface{}{"path": filePath})
			continue
		}
		b, err := os.ReadFile(filePath)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read file in git repository", err.Error())
			return
		}
		repositoryFiles[k] = string(b)
	}

	kubeClient, err := r.prd.GetKubernetesClient()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes Client", err.Error())
		return
	}

	// Check cluster access and kubeconfig permissions
	if err := isKubernetesReady(ctx, kubeClient); err != nil {
		resp.Diagnostics.AddError("Kubernetes cluster", err.Error())
		return
	}

	// Detect drift for the Flux installation in the cluster.
	ready, err := isFluxReady(ctx, kubeClient, data)
	if !ready {
		// reset the gotk_sync.yaml file content to simulate a Git drift which will trigger a redeployment
		syncOpts := sync.MakeDefaultOptions()
		syncPath := filepath.Join(data.Path.ValueString(), data.Namespace.ValueString(), syncOpts.ManifestFile)
		repositoryFiles[syncPath] = ""
		warnDetails := ""
		if err != nil {
			warnDetails = err.Error()
		}
		resp.Diagnostics.AddWarning("Flux controllers are not healthy and will be redeployed", warnDetails)
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

// Update pushes the Flux manifests in the Git repository and applies the changes on the cluster.
func (r bootstrapGitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.prd == nil {
		resp.Diagnostics.AddError(missingConfiguration, bootstrapGitResourceMissingConfigError)
	}

	var data bootstrapGitResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := data.Timeouts.Update(ctx, defaultUpdateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	previousRepositoryFiles := types.MapNull(types.StringType)
	diags = req.State.GetAttribute(ctx, path.Root("repository_files"), &previousRepositoryFiles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	repositoryFiles := map[string]string{}
	diags = data.RepositoryFiles.ElementsAs(ctx, &repositoryFiles, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Sync Git repository with Terraform state.
	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		gitClient, err := r.prd.CloneRepository(ctx)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		defer os.RemoveAll(gitClient.Path())

		// Files should be removed if they are present in the state but not the plan.
		for k := range previousRepositoryFiles.Elements() {
			_, ok := data.RepositoryFiles.Elements()[k]
			if ok {
				continue
			}
			filePath := filepath.Join(gitClient.Path(), k)
			_, err := os.Stat(filePath)
			if errors.Is(err, os.ErrNotExist) {
				tflog.Debug(ctx, "Skipping removing no longer tracked file as it does not exist", map[string]interface{}{"path": filePath})
				continue
			}
			if err != nil {
				retry.NonRetryableError(fmt.Errorf("could not stat no longer tracked file: %w", err))
			}
			err = os.Remove(filePath)
			if err != nil {
				return retry.NonRetryableError(fmt.Errorf("could not remove no longer tracked file: %w", err))
			}
		}

		// Write expected file contents to repo.
		files := map[string]io.Reader{}
		for k, v := range repositoryFiles {
			files[k] = strings.NewReader(v)
		}
		commit, signer, err := r.prd.CreateCommit("Update Flux manifests")
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("unable to create commit: %w", err))
		}
		_, err = gitClient.Commit(commit, signer, repository.WithFiles(files))
		if err != nil && !errors.Is(err, git.ErrNoStagedFiles) {
			return retry.NonRetryableError(fmt.Errorf("unable to commit updated files: %w", err))
		}
		// Skip pushing if no changes have been made.
		if err != nil {
			return nil
		}
		err = gitClient.Push(ctx, repository.PushConfig{})
		if err != nil {
			return retry.RetryableError(fmt.Errorf("unable to push updated manifests: %w", err))
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Could not update Flux manifests in Git", err.Error())
	} else {
		// Sync Flux installation with Git state.

		installOpts := getInstallOptions(data)
		syncOpts := getSyncOptions(data, r.prd.GetRepositoryURL(), r.prd.git.Branch.ValueString())
		var secretOpts sourcesecret.Options
		if data.DisableSecretCreation.ValueBool() {
			secretOpts = sourcesecret.Options{
				Name:      data.SecretName.ValueString(),
				Namespace: data.Namespace.ValueString(),
			}
		} else {
			secretOpts, err = r.prd.GetSecretOptions(data.SecretName.ValueString(), data.Namespace.ValueString(), data.Path.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Could not get secret options", err.Error())
				return
			}
		}

		tmpDir, err := manifestgen.MkdirTempAbs("", "flux-bootstrap-")
		if err != nil {
			resp.Diagnostics.AddError("could not create temporary working directory for git repository", err.Error())
		}
		defer os.RemoveAll(tmpDir)

		bootstrapProvider, err := r.prd.GetBootstrapProvider(tmpDir)
		if err != nil {
			resp.Diagnostics.AddError("Bootstrap Provider", err.Error())
			return
		}

		manifestsBase := ""
		if data.EmbeddedManifests.ValueBool() {
			manifestsBase = EmbeddedManifests
		}
		err = bootstrap.Run(ctx, bootstrapProvider, manifestsBase, installOpts, secretOpts, syncOpts, 2*time.Second, timeout)
		if err != nil {
			resp.Diagnostics.AddError("Bootstrap run error", err.Error())
			return
		}
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Delete removes the Flux components from the cluster and the manifests from the Git repository.
func (r bootstrapGitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.prd == nil {
		resp.Diagnostics.AddError(missingConfiguration, bootstrapGitResourceMissingConfigError)
	}

	var data bootstrapGitResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := data.Timeouts.Delete(ctx, defaultDeleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	kubeClient, err := r.prd.GetKubernetesClient()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes Client", err.Error())
		tflog.Error(ctx, "Unable to get Kubernetes client", map[string]interface{}{})
		return
	}

	err = uninstall.Components(ctx, log.NopLogger{}, kubeClient, data.Namespace.ValueString(), false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove Flux components", err.Error())
		tflog.Debug(ctx, "Unable to remove Flux components", map[string]interface{}{})
	}
	err = uninstall.Finalizers(ctx, log.NopLogger{}, kubeClient, false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove finalizers", err.Error())
		tflog.Debug(ctx, "Unable to remove finalizers", map[string]interface{}{})
	}
	err = uninstall.CustomResourceDefinitions(ctx, log.NopLogger{}, kubeClient, false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove CRDs", err.Error())
		tflog.Debug(ctx, "Unable to remove CRDs", map[string]interface{}{})
	}

	// Only remove namespace if not keeping it.
	if data.KeepNamespace.ValueBool() {
		tflog.Debug(ctx, fmt.Sprintf("The keep_namespace variable was set to true. Skipping removal of %s namespace.", data.Namespace.ValueString()), map[string]interface{}{})
	} else {
		err = uninstall.Namespace(ctx, log.NopLogger{}, kubeClient, data.Namespace.ValueString(), false)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Unable to remove %s namespace.", data.Namespace.ValueString()), err.Error())
			tflog.Debug(ctx, fmt.Sprintf("Unable to remove %s namespace.", data.Namespace.ValueString()), map[string]interface{}{})
		}
	}

	if !(data.DeleteGitManifests.IsNull() || data.DeleteGitManifests.ValueBool()) {
		tflog.Debug(ctx, "Skipping git repository removal", map[string]interface{}{})
		return
	}

	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		gitClient, err := r.prd.CloneRepository(ctx)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		defer os.RemoveAll(gitClient.Path())

		// Remove all tracked files from git.
		for k := range data.RepositoryFiles.Elements() {
			path := filepath.Join(gitClient.Path(), k)
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				tflog.Debug(ctx, "Skipping file removal as the file does not exist", map[string]interface{}{"path": path})
				continue
			}
			err := os.Remove(path)
			if err != nil {
				return retry.NonRetryableError(fmt.Errorf("could not remove file from git repository: %w", err))
			}
		}
		// TODO: If no files are removed we should not commit anything.
		commit, signer, err := r.prd.CreateCommit("Uninstall Flux")
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("unable to create commit: %w", err))
		}

		// TODO: If all files are removed from the repository delete will fail. This needs a test and to be fixed.
		_, err = gitClient.Commit(commit, signer)
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("unable to commit removed file(s): %w", err))
		}

		err = gitClient.Push(ctx, repository.PushConfig{})
		if err != nil {
			return retry.RetryableError(fmt.Errorf("unable to push removed file(s): %w", err))
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Could not delete Flux configuration from Git repository.", err.Error())
	}
}

// ImportState scans the cluster for the Flux components configuration and imports it into the Terraform state.
func (r *bootstrapGitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if r.prd == nil {
		resp.Diagnostics.AddError(missingConfiguration, bootstrapGitResourceMissingConfigError)
	}

	kubeClient, err := r.prd.GetKubernetesClient()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes Client", err.Error())
		return
	}

	data := bootstrapGitResourceData{}
	data.Timeouts = timeouts.Value{
		Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"delete": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
		}),
	}
	data.ID = types.StringValue(req.ID)
	data.Namespace = data.ID

	ready, err := isFluxReady(ctx, kubeClient, data)
	if err != nil {
		resp.Diagnostics.AddError("Could not check Flux readiness", err.Error())
		return
	}
	if !ready {
		resp.Diagnostics.AddError("Flux is not ready", "Flux Kustomization is failing")
		return
	}

	// Set values that cant be null.
	data.TolerationKeys = types.SetNull(types.StringType)

	// Stub keep namespace and delete git manifests to their defaults.
	data.KeepNamespace = types.BoolValue(false)
	data.DeleteGitManifests = types.BoolValue(true)

	// Get Network NetworkPolicy.
	networkPolicy := networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-webhooks",
			Namespace: data.Namespace.ValueString(),
		},
	}
	err = kubeClient.Get(ctx, client.ObjectKeyFromObject(&networkPolicy), &networkPolicy)
	if err != nil && !k8serrors.IsNotFound(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not get NetworkPolicy %s/%s", networkPolicy.Namespace, networkPolicy.Name), err.Error())
		return
	}
	data.NetworkPolicy = types.BoolValue(true)
	if err != nil && k8serrors.IsNotFound(err) {
		data.NetworkPolicy = types.BoolValue(false)
	}

	// Get values from kustomize-controller Deployment.
	kustomizeDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kustomize-controller",
			Namespace: data.Namespace.ValueString(),
		},
	}
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(&kustomizeDeployment), &kustomizeDeployment); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not get Deployment %s/%s", kustomizeDeployment.Namespace, kustomizeDeployment.Name), err.Error())
		return
	}
	managerContainer, err := utils.GetContainer(kustomizeDeployment.Spec.Template.Spec.Containers, "manager")
	if err != nil {
		resp.Diagnostics.AddError("Could not get manager container", err.Error())
		return
	}

	// Get Flux Version being used.
	version, ok := kustomizeDeployment.Labels["app.kubernetes.io/version"]
	if !ok {
		resp.Diagnostics.AddError("Version label not found", "Label is not present in kustomize-controller Deployment")
		return
	}
	data.Version = types.StringValue(version)

	// Get Image Registry.
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

	// Get the toleration keys.
	tolerationKeys := []string{}
	for _, toleration := range kustomizeDeployment.Spec.Template.Spec.Tolerations {
		tolerationKeys = append(tolerationKeys, toleration.Key)
	}
	tolerationKeysSet, diags := types.SetValueFrom(ctx, types.StringType, tolerationKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.TolerationKeys = tolerationKeysSet

	// Get image pull secrets.
	data.ImagePullSecret = types.StringNull()
	if len(kustomizeDeployment.Spec.Template.Spec.ImagePullSecrets) > 0 {
		data.ImagePullSecret = types.StringValue(kustomizeDeployment.Spec.Template.Spec.ImagePullSecrets[0].Name)
	}

	// Get if watching all namespace.
	value, err := utils.GetArgValue(managerContainer, "--watch-all-namespaces")
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
	value, err = utils.GetArgValue(managerContainer, "--log-level")
	if err != nil {
		resp.Diagnostics.AddError("Could not get arg", err.Error())
		return
	}
	data.LogLevel = types.StringValue(value)

	// Get cluster domain
	value, err = utils.GetArgValue(managerContainer, "--events-addr")
	if err != nil {
		resp.Diagnostics.AddError("Could not get arg", err.Error())
		return
	}
	eventsUrl, err := url.Parse(value)
	if err != nil {
		resp.Diagnostics.AddError("Could not parse events address", err.Error())
		return
	}
	// TODO: Probably smarter to remove what we know comes before the cluster domain and remove that.
	host := strings.TrimSuffix(eventsUrl.Host, ".")
	c := strings.Split(host, ".")
	clusterDomain := strings.Join(c[len(c)-2:], ".")
	data.ClusterDomain = types.StringValue(clusterDomain)

	// Get values from flux-system GitRepository.
	gitRepository := sourcev1.GitRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Namespace.ValueString(),
			Namespace: data.Namespace.ValueString(),
		},
	}
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(&gitRepository), &gitRepository); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not get GitRepository %s/%s", gitRepository.Namespace, gitRepository.Name), err.Error())
		return
	}
	data.SecretName = types.StringValue(gitRepository.Spec.SecretRef.Name)
	data.Interval = customtypes.DurationValue(gitRepository.Spec.Interval.Duration)

	// Get values from flux-system Kustomization.
	kustomization := kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Namespace.ValueString(),
			Namespace: data.Namespace.ValueString(),
		},
	}
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(&kustomization), &kustomization); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not get Kustomization %s/%s", kustomization.Namespace, kustomization.Name), err.Error())
		return
	}
	// Only set path value if path is something other than nil. This is to be consistent with the default value.
	data.Path = types.StringNull()
	syncPath := strings.TrimPrefix(kustomization.Spec.Path, "./")
	if syncPath != "" {
		data.Path = types.StringValue(syncPath)
	}

	// Check which components are present and which are not.
	components := []attr.Value{}
	for _, c := range install.MakeDefaultOptions().Components {
		dep := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c,
				Namespace: data.Namespace.ValueString(),
			},
		}
		err := kubeClient.Get(ctx, client.ObjectKeyFromObject(&dep), &dep)
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
		err := kubeClient.Get(ctx, client.ObjectKeyFromObject(&dep), &dep)
		if err != nil && k8serrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Could not get Deployment %s/%s", dep.Namespace, dep.Name), err.Error())
			return
		}
		componentsExtra = append(componentsExtra, types.StringValue(c))
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

	// Set expected repository files.
	repositoryFiles, err := getExpectedRepositoryFiles(data, r.prd.GetRepositoryURL(), r.prd.git.Branch.ValueString())
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

	baseURL := data.ManifestsPath.ValueString()
	if baseURL == "" {
		baseURL = install.MakeDefaultOptions().BaseURL
	}

	installOptions := install.Options{
		BaseURL:                baseURL,
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

func getSyncOptions(data bootstrapGitResourceData, url *url.URL, branch string) sync.Options {
	syncOpts := sync.Options{
		Interval:          data.Interval.ValueDuration(),
		Name:              data.Namespace.ValueString(),
		Namespace:         data.Namespace.ValueString(),
		URL:               url.String(),
		Branch:            branch,
		Secret:            data.SecretName.ValueString(),
		TargetPath:        data.Path.ValueString(),
		ManifestFile:      sync.MakeDefaultOptions().ManifestFile,
		RecurseSubmodules: data.RecurseSubmodules.ValueBool(),
	}
	return syncOpts
}

func getExpectedRepositoryFiles(data bootstrapGitResourceData, url *url.URL, branch string) (map[string]string, error) {
	repositoryFiles := map[string]string{}
	installOpts := getInstallOptions(data)
	manifestsBase := ""
	if data.EmbeddedManifests.ValueBool() {
		manifestsBase = EmbeddedManifests
	}

	installManifests, err := install.Generate(installOpts, manifestsBase)
	if err != nil {
		return nil, fmt.Errorf("could not generate install manifests: %w", err)
	}

	repositoryFiles[installManifests.Path] = installManifests.Content

	syncOpts := getSyncOptions(data, url, branch)
	syncManifests, err := sync.Generate(syncOpts)
	if err != nil {
		return nil, fmt.Errorf("could not generate sync manifests: %w", err)
	}

	repositoryFiles[syncManifests.Path] = syncManifests.Content
	repositoryFiles[filepath.Join(data.Path.ValueString(), data.Namespace.ValueString(), konfig.DefaultKustomizationFileName())] = getKustomizationFile(data)

	return repositoryFiles, nil
}

// isKubernetesReady checks if the Kubernetes API is accessible
// and if the user has the necessary permissions.
func isKubernetesReady(ctx context.Context, kubeClient client.Client) error {
	var list apiextensionsv1.CustomResourceDefinitionList
	selector := client.MatchingLabels{manifestgen.PartOfLabelKey: manifestgen.PartOfLabelValue}
	if err := kubeClient.List(ctx, &list, client.InNamespace(""), selector); err != nil {
		return err
	}
	return nil
}

// isFluxReady checks if the Flux sync objects are present and ready.
func isFluxReady(ctx context.Context, kubeClient client.Client, data bootstrapGitResourceData) (bool, error) {
	syncName := apitypes.NamespacedName{
		Namespace: data.Namespace.ValueString(),
		Name:      data.Namespace.ValueString(),
	}

	rootSource := &sourcev1.GitRepository{}
	if err := kubeClient.Get(ctx, syncName, rootSource); err != nil {
		return false, err
	}
	if conditions.IsFalse(rootSource, meta.ReadyCondition) {
		return false, errors.New(conditions.GetMessage(rootSource, meta.ReadyCondition))
	}

	rootSync := &kustomizev1.Kustomization{}
	if err := kubeClient.Get(ctx, syncName, rootSync); err != nil {
		return false, err
	}
	if conditions.IsFalse(rootSync, meta.ReadyCondition) {
		conditions.GetMessage(rootSync, meta.ReadyCondition)
		return false, errors.New(conditions.GetMessage(rootSync, meta.ReadyCondition))
	}

	return true, nil
}
