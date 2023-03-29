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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

type bootstrapGitResourceData struct {
	ID                    types.String         `tfsdk:"id"`
	Version               types.String         `tfsdk:"version"`
	Path                  types.String         `tfsdk:"path"`
	ClusterDomain         types.String         `tfsdk:"cluster_domain"`
	Components            types.Set            `tfsdk:"components"`
	ComponentsExtra       types.Set            `tfsdk:"components_extra"`
	ImagePullSecret       types.String         `tfsdk:"image_pull_secret"`
	LogLevel              types.String         `tfsdk:"log_level"`
	Namespace             types.String         `tfsdk:"namespace"`
	NetworkPolicy         types.Bool           `tfsdk:"network_policy"`
	Registry              customtypes.URL      `tfsdk:"registry"`
	TolerationKeys        types.Set            `tfsdk:"toleration_keys"`
	WatchAllNamespaces    types.Bool           `tfsdk:"watch_all_namespaces"`
	Interval              customtypes.Duration `tfsdk:"interval"`
	SecretName            types.String         `tfsdk:"secret_name"`
	DisableSecretCreation types.Bool           `tfsdk:"disable_secret_creation"`
	RecurseSubmodules     types.Bool           `tfsdk:"recurse_submodules"`
	KustomizationOverride types.String         `tfsdk:"kustomization_override"`
	RepositoryFiles       types.Map            `tfsdk:"repository_files"`
	Timeouts              timeouts.Value       `tfsdk:"timeouts"`
}

// Ensure provider defined types fully satisfy framework interfaces
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
			"id": schema.StringAttribute{
				Computed: true,
			},
			"version": schema.StringAttribute{
				Description: fmt.Sprintf("Flux version. Defaults to `%s`.", utils.DefaultFluxVersion),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(utils.DefaultFluxVersion),
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("(latest|^v.*)"), "must either be latest or start with 'v'"),
				},
			},
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
			"image_pull_secret": schema.StringAttribute{
				Description: "Kubernetes secret name used for pulling the toolkit images from a private registry.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(rfc1123DomainRegex), rfc1123DomainError),
					stringvalidator.LengthAtMost(253),
				},
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
			"namespace": schema.StringAttribute{
				Description: fmt.Sprintf("The namespace scope for install manifests. Defaults to `%s`.", defaultOpts.Namespace),
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
			"registry": schema.StringAttribute{
				CustomType:  customtypes.URLType{},
				Description: fmt.Sprintf("Container registry where the toolkit images are published. Defaults to `%s`.", defaultOpts.Registry),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultOpts.Registry),
			},
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
			"watch_all_namespaces": schema.BoolAttribute{
				Description: fmt.Sprintf("If true watch for custom resources in all namespaces. Defaults to `%v`.", defaultOpts.WatchAllNamespaces),
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(defaultOpts.WatchAllNamespaces),
			},
			"interval": schema.StringAttribute{
				CustomType:  customtypes.DurationType{},
				Description: fmt.Sprintf("Interval at which to reconcile from bootstrap repository. Defaults to `%s`.", time.Minute.String()),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(time.Minute.String()),
			},
			"path": schema.StringAttribute{
				Description: "Path relative to the repository root, when specified the cluster sync will be scoped to this path.",
				Optional:    true,
			},
			"recurse_submodules": schema.BoolAttribute{
				Description: "Configures the GitRepository source to initialize and include Git submodules in the artifact it produces.",
				Optional:    true,
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
			"disable_secret_creation": schema.BoolAttribute{
				Description: "Use the existing secret for flux controller and don't create one from bootstrap",
				Optional:    true,
			},
			"kustomization_override": schema.StringAttribute{
				Description: "Kustomization to override configuration set by default.",
				Optional:    true,
				Validators:  []validator.String{validators.KustomizationOverride()},
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

	// Write expected repository files
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
	// This has to be set here, probably a bug in the SDK
	diags = resp.Plan.SetAttribute(ctx, path.Root("id"), data.Namespace)
	resp.Diagnostics.Append(diags...)
}

// TODO: If kustomization file exists and not all resource files exist bootstrap will fail. This is because kustomize build is run.
func (r *bootstrapGitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

	gitClient, err := r.prd.GetGitClient(ctx)
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
	b, err := bootstrap.NewPlainGitProvider(gitClient, kubeClient, bootstrapOpts...)
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
		commit, signer, err := r.prd.CreateCommit("Add kustomize override")
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
	err = bootstrap.Run(ctx, b, manifestsBase, installOpts, secretOpts, syncOpts, 2*time.Second, timeout)
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
// TODO: Resources in the cluster should be verified to exist. If not resource id should be set to nil. This is to detect changing clusters.
func (r *bootstrapGitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

	gitClient, err := r.prd.GetGitClient(ctx)
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

	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		gitClient, err := r.prd.GetGitClient(ctx)
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
			path := filepath.Join(gitClient.Path(), k)
			_, err := os.Stat(path)
			if errors.Is(err, os.ErrNotExist) {
				tflog.Debug(ctx, "Skipping removing no longer tracked file as it does not exist", map[string]interface{}{"path": path})
				continue
			}
			if err != nil {
				retry.NonRetryableError(fmt.Errorf("Could not stat no longer tracked file: %w", err))
			}
			err = os.Remove(path)
			if err != nil {
				return retry.NonRetryableError(fmt.Errorf("Could not remove no longer tracked file: %w", err))
			}
		}

		// Write expected file contents to repo
		files := map[string]io.Reader{}
		for k, v := range repositoryFiles {
			files[k] = strings.NewReader(v)
		}
		commit, signer, err := r.prd.CreateCommit("Update Flux")
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("Unable to create commit: %w", err))
		}
		_, err = gitClient.Commit(commit, signer, repository.WithFiles(files))
		if err != nil && !errors.Is(err, git.ErrNoStagedFiles) {
			return retry.NonRetryableError(fmt.Errorf("Unable to commit updated files: %w", err))
		}
		// Skip pushing if no changes have been made
		if err != nil {
			return nil
		}
		err = gitClient.Push(ctx)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("Unable to push file update: %w", err))
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Could not update Flux", err.Error())
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r bootstrapGitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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
		return
	}
	// TODO: Uninstall fails when flux-system namespace does not exist
	err = uninstall.Components(ctx, log.NopLogger{}, kubeClient, data.Namespace.ValueString(), false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove Flux components", err.Error())
		return
	}
	err = uninstall.Finalizers(ctx, log.NopLogger{}, kubeClient, false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove finalizers", err.Error())
		return
	}
	err = uninstall.CustomResourceDefinitions(ctx, log.NopLogger{}, kubeClient, false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove CRDs", err.Error())
		return
	}
	err = uninstall.Namespace(ctx, log.NopLogger{}, kubeClient, data.Namespace.ValueString(), false)
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove namespace", err.Error())
		return
	}

	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		gitClient, err := r.prd.GetGitClient(ctx)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		defer os.RemoveAll(gitClient.Path())

		// Remove all tracked files from git
		for k := range data.RepositoryFiles.Elements() {
			path := filepath.Join(gitClient.Path(), k)
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				tflog.Debug(ctx, "Skipping file removal as the file does not exist", map[string]interface{}{"path": path})
				continue
			}
			err := os.Remove(path)
			if err != nil {
				return retry.NonRetryableError(fmt.Errorf("Could not remove file from git repository: %w", err))
			}
		}
		// TODO: If no files are removed we should not commit anything.
		commit, signer, err := r.prd.CreateCommit("Uninstall Flux")
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("Unable to create commit: %w", err))
		}
		// TODO: If all files are removed from the repository delete will fail. This needs a test and to be fixed.
		_, err = gitClient.Commit(commit, signer)
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("Unable to commit removed file(s): %w", err))
		}
		err = gitClient.Push(ctx)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("Unable to psuh removed file(s): %w", err))
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Could not delete Flux", err.Error())
	}
}

// TODO: Validate Flux installation before proceeding with import
func (r *bootstrapGitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

	// Set values that cant be null
	data.TolerationKeys = types.SetNull(types.StringType)

	// Get Network NetworkPolicy
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

	// Get values from kustomize-controller Deployment
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

	// Get the toleration keys
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

	// Get image pull secrets
	data.ImagePullSecret = types.StringNull()
	if len(kustomizeDeployment.Spec.Template.Spec.ImagePullSecrets) > 0 {
		data.ImagePullSecret = types.StringValue(kustomizeDeployment.Spec.Template.Spec.ImagePullSecrets[0].Name)
	}

	// Get if watching all namespace
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
	if err := kubeClient.Get(ctx, client.ObjectKeyFromObject(&gitRepository), &gitRepository); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not get GitRepository %s/%s", gitRepository.Namespace, gitRepository.Name), err.Error())
		return
	}
	data.SecretName = types.StringValue(gitRepository.Spec.SecretRef.Name)
	data.Interval = customtypes.DurationValue(gitRepository.Spec.Interval.Duration)

	// Get values from flux-system Kustomization
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
	path := strings.TrimPrefix(kustomization.Spec.Path, "./")
	if path != "" {
		data.Path = types.StringValue(path)
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

	// Set expected repository files
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
	installManifests, err := install.Generate(installOpts, "")
	if err != nil {
		return nil, fmt.Errorf("Could not generate install manifests: %w", err)
	}
	repositoryFiles[installManifests.Path] = installManifests.Content
	syncOpts := getSyncOptions(data, url, branch)
	syncManifests, err := sync.Generate(syncOpts)
	if err != nil {
		return nil, fmt.Errorf("Could not generate sync manifests: %w", err)
	}
	repositoryFiles[syncManifests.Path] = syncManifests.Content
	repositoryFiles[filepath.Join(data.Path.ValueString(), data.Namespace.ValueString(), konfig.DefaultKustomizationFileName())] = getKustomizationFile(data)
	return repositoryFiles, nil
}
