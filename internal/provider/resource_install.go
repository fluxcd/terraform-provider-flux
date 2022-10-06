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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	//"go.uber.org/multierr"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/fluxcd/flux2/pkg/manifestgen"
	"github.com/fluxcd/flux2/pkg/manifestgen/install"
	"github.com/fluxcd/flux2/pkg/status"
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	autov1 "github.com/fluxcd/image-automation-controller/api/v1beta1"
	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	runclient "github.com/fluxcd/pkg/runtime/client"
	"github.com/fluxcd/pkg/ssa"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/fluxcd/terraform-provider-flux/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ provider.ResourceType = installResourceType{}
var _ resource.Resource = installResource{}
var _ resource.ResourceWithImportState = installResource{}

type installResourceType struct{}

func (t installResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	defaultOpts := install.MakeDefaultOptions()
	defaultComponents := stringListToSetString(defaultOpts.Components)

	return tfsdk.Schema{
		MarkdownDescription: "Installs Flux into a Kubernetes cluster.",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:     types.StringType,
				Computed: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"cluster_domain": {
				Description: "The internal cluster domain.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
				// Might force new
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown(), NewDefaultModifier(types.String{Value: defaultOpts.ClusterDomain})},
			},
			"components": {
				Description: "Toolkit components to include in the install manifests.",
				Type: types.SetType{
					ElemType: types.StringType,
				},
				Optional:      true,
				Computed:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown(), NewDefaultModifier(defaultComponents)},
			},
			"components_extra": {
				Description: "List of extra components to include in the install manifests.",
				Type: types.SetType{
					ElemType: types.StringType,
				},
				Optional: true,
			},
			"image_pull_secrets": {
				Description:   "Kubernetes secret name used for pulling the toolkit images from a private registry.",
				Type:          types.StringType,
				Optional:      true,
				Computed:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown(), NewDefaultModifier(types.String{Value: defaultOpts.ImagePullSecret})},
			},
			"log_level": {
				Description:   "Log level for toolkit components.",
				Type:          types.StringType,
				Optional:      true,
				Computed:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown(), NewDefaultModifier(types.String{Value: defaultOpts.LogLevel})},
				Validators:    []tfsdk.AttributeValidator{stringvalidator.OneOf("info", "debug", "error")},
			},
			"namespace": {
				Description: "The namespace scope for install manifests.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
				// Should force new
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown(), NewDefaultModifier(types.String{Value: defaultOpts.Namespace})},
			},
			"network_policy": {
				Description:   "Deny ingress access to the toolkit controllers from other namespaces using network policies.",
				Type:          types.BoolType,
				Optional:      true,
				Computed:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown(), NewDefaultModifier(types.Bool{Value: defaultOpts.NetworkPolicy})},
			},
			"registry": {
				Description:   "Container registry where the toolkit images are published.",
				Type:          types.StringType,
				Optional:      true,
				Computed:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown(), NewDefaultModifier(types.String{Value: defaultOpts.Registry})},
			},
			"toleration_keys": {
				Description: "List of toleration keys used to schedule the components pods onto nodes with matching taints.",
				Type: types.SetType{
					ElemType: types.StringType,
				},
				Optional: true,
			},
			"version": {
				Description:   "Flux version.",
				Type:          types.StringType,
				Optional:      true,
				Computed:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown(), NewDefaultModifier(types.String{Value: "v0.32.0"})},
				Validators:    []tfsdk.AttributeValidator{stringvalidator.RegexMatches(regexp.MustCompile("(latest|^v.*)"), "must either be latest or start with 'v'")},
			},
			"watch_all_namespaces": {
				Description:   "If true watch for custom resources in all namespaces.",
				Type:          types.BoolType,
				Optional:      true,
				Computed:      true,
				PlanModifiers: tfsdk.AttributePlanModifiers{resource.UseStateForUnknown(), NewDefaultModifier(types.Bool{Value: defaultOpts.WatchAllNamespaces})},
			},
			"cluster_yaml": {
				Description: "YAML applied to the cluster.",
				Computed:    true,
				Type: types.MapType{
					ElemType: types.StringType,
				},
			},
		},
	}, nil
}

func (t installResourceType) NewResource(ctx context.Context, in provider.Provider) (resource.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)
	return installResource{
		provider: provider,
	}, diags
}

type installResourceData struct {
	Id                 types.String `tfsdk:"id"`
	ClusterDomain      string       `tfsdk:"cluster_domain"`
	Components         []string     `tfsdk:"components"`
	ComponentsExtra    []string     `tfsdk:"components_extra"`
	ImagePullSecrets   string       `tfsdk:"image_pull_secrets"`
	LogLevel           string       `tfsdk:"log_level"`
	Namespace          string       `tfsdk:"namespace"`
	NetworkPolicy      bool         `tfsdk:"network_policy"`
	Registry           string       `tfsdk:"registry"`
	TolerationKeys     []string     `tfsdk:"toleration_keys"`
	Version            string       `tfsdk:"version"`
	WatchAllNamespaces bool         `tfsdk:"watch_all_namespaces"`
	ClusterYaml        types.Map    `tfsdk:"cluster_yaml"`
}

type installResource struct {
	provider fluxProvider
}

func (r installResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data installResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := getInstallOptions(data)
	objs, err := getOjbects(opts)
	if err != nil {
		resp.Diagnostics.AddError("Flux Error", err.Error())
		return
	}

	// Apply objects to cluster
	_, err = utils.Apply(ctx, &r.provider.rcg, &runclient.Options{}, objs)
	if err != nil {
		resp.Diagnostics.AddError("Flux Error", err.Error())
		return
	}

	// Get in cluster YAML of applied objects
	// TODO: Could be made simpler if apply returned the in cluster YAML
	data.ClusterYaml = types.Map{
		ElemType: types.StringType,
		Elems:    map[string]attr.Value{},
	}
	for _, obj := range objs {
		_ = r.provider.manager.Client().Get(ctx, client.ObjectKeyFromObject(obj), obj)
		if obj == nil {
			continue
		}
		d, err := marshalObject(obj)
		if err != nil {
			resp.Diagnostics.AddError("Flux Error", err.Error())
			return
		}
		data.ClusterYaml.Elems[ssa.FmtUnstructured(obj)] = types.String{Value: d}
	}

	// Wait for Flux to be deployed
	kubeConfig, err := utils.KubeConfig(&r.provider.rcg, &runclient.Options{})
	if err != nil {
		resp.Diagnostics.AddError("Flux Error", err.Error())
		return
	}
	timeout := 5 * time.Minute
	statusChecker, err := status.NewStatusChecker(kubeConfig, 5*time.Second, timeout, utils.DiscardLogger{})
	if err != nil {
		resp.Diagnostics.AddError("Flux Error", err.Error())
		return
	}
	componentRefs, err := utils.BuildComponentObjectRefs(opts.Namespace, opts.Components...)
	if err != nil {
		resp.Diagnostics.AddError("Flux Error", err.Error())
		return
	}
	if err := statusChecker.Assess(componentRefs...); err != nil {
		resp.Diagnostics.AddError("Flux Error", err.Error())
		return
	}

	// Flux has been sucessfully installed
	data.Id = types.String{Value: data.Namespace}
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r installResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data installResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update tracked Kubernetes resources with current in cluster YAML.
	// Resources that do not exist in cluster will be removed from state.
	// TODO: Dry run should be a better option here to determine if YAML has actually changed.
	// In cluster YAML may have actually changed but it does not matter if the fields are not
	// managed by this provider. If it has not changed the previous states YAML should be kept.
	clusterYaml := types.Map{
		ElemType: types.StringType,
		Elems:    map[string]attr.Value{},
	}
	for k, v := range data.ClusterYaml.Elems {
		obj, err := ssa.ReadObject(strings.NewReader(v.(types.String).Value))
		if err != nil {
			resp.Diagnostics.AddError("Flux Error", err.Error())
			return
		}
		_ = r.provider.manager.Client().Get(ctx, client.ObjectKeyFromObject(obj), obj)
		if obj == nil {
			continue
		}
		d, err := marshalObject(obj)
		if err != nil {
			resp.Diagnostics.AddError("Flux Error", err.Error())
			return
		}
		clusterYaml.Elems[k] = types.String{Value: d}
	}

	data.ClusterYaml = clusterYaml
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r installResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data installResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := getInstallOptions(data)
	objs, err := getOjbects(opts)
	if err != nil {
		resp.Diagnostics.AddError("Flux Error", err.Error())
		return
	}
	_, err = utils.Apply(ctx, &r.provider.rcg, &runclient.Options{}, objs)
	if err != nil {
		resp.Diagnostics.AddError("Flux Error", err.Error())
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r installResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data installResourceData
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	kubeClient, err := utils.KubeClient(&r.provider.rcg, &runclient.Options{})
	if err != nil {
		resp.Diagnostics.AddError("Flux Error", err.Error())
		return
	}
	err = uninstallFlux(ctx, kubeClient, data.Namespace)
	if err != nil {
		resp.Diagnostics.AddError("Flux Error", err.Error())
		return
	}
}

func (r installResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
  // TODO: Namespace should be enough to track the flux installation as only one can exist per namespace
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r installResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() {
		return
	}

	var data installResourceData
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := getInstallOptions(data)
	objs, err := getOjbects(opts)
	if err != nil {
		resp.Diagnostics.AddError("Flux Error", err.Error())
		return
	}

	clusterYaml := types.Map{
		ElemType: types.StringType,
		Elems:    map[string]attr.Value{},
	}
	for _, obj := range objs {
		opts := []client.PatchOption{
			client.DryRunAll,
			client.ForceOwnership,
			client.FieldOwner("flux"),
		}
		dry := obj.DeepCopy()
		err := r.provider.manager.Client().Patch(ctx, dry, client.Apply, opts...)
		if err != nil {
			resp.Diagnostics.AddError("Flux Error", err.Error())
			return
		}
		d, err := marshalObject(dry)
		if err != nil {
			resp.Diagnostics.AddError("Flux Error", err.Error())
			return
		}
		clusterYaml.Elems[ssa.FmtUnstructured(dry)] = types.String{Value: d}
	}

	data.ClusterYaml = clusterYaml
	diags = resp.Plan.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func getInstallOptions(data installResourceData) install.Options {
	components := append(data.Components, data.ComponentsExtra...)
	opts := install.Options{
		BaseURL:                install.MakeDefaultOptions().BaseURL,
		Version:                data.Version,
		Namespace:              data.Namespace,
		Components:             components,
		Registry:               data.Registry,
		ImagePullSecret:        data.ImagePullSecrets,
		WatchAllNamespaces:     data.WatchAllNamespaces,
		NetworkPolicy:          data.NetworkPolicy,
		LogLevel:               data.LogLevel,
		NotificationController: install.MakeDefaultOptions().NotificationController,
		ManifestFile:           fmt.Sprintf("%s.yaml", data.Namespace),
		Timeout:                install.MakeDefaultOptions().Timeout,
		ClusterDomain:          data.ClusterDomain,
		TolerationKeys:         data.TolerationKeys,
	}
	return opts
}

func getOjbects(opts install.Options) ([]*unstructured.Unstructured, error) {
	tmpDir, err := manifestgen.MkdirTempAbs("", "flux-system")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	manifest, err := install.Generate(opts, "")
	if err != nil {
		return nil, err
	}
	if _, err := manifest.WriteFile(tmpDir); err != nil {
		return nil, err
	}
	objs, err := utils.ReadObjects(tmpDir, filepath.Join(tmpDir, manifest.Path))
	if err != nil {
		return nil, err
	}
	return objs, nil
}

func marshalObject(obj *unstructured.Unstructured) (string, error) {
	obj = obj.DeepCopy()
	unstructured.RemoveNestedField(obj.Object, "metadata", "managedFields")
	unstructured.RemoveNestedField(obj.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(obj.Object, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(obj.Object, "metadata", "uid")
	unstructured.RemoveNestedField(obj.Object, "metadata", "generation")
	unstructured.RemoveNestedField(obj.Object, "metadata", "annotations", "deployment.kubernetes.io/revision")
	unstructured.RemoveNestedField(obj.Object, "status")
	b, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func stringListToSetString(list []string) types.Set {
	set := types.Set{
		ElemType: types.StringType,
		Elems:    []attr.Value{},
	}
	for _, item := range list {
		set.Elems = append(set.Elems, types.String{Value: item})
	}
	return set
}

func uninstallFlux(ctx context.Context, kubeClient client.Client, namespace string) error {
	err := uninstallComponents(ctx, kubeClient, namespace)
	if err != nil {
		return err
	}
	err = uninstallFinalizers(ctx, kubeClient)
	if err != nil {
		return err
	}
	err = uninstallCustomResourceDefinitions(ctx, kubeClient)
	if err != nil {
		return err
	}
	err = uninstallNamespace(ctx, kubeClient, namespace)
	if err != nil {
		return err
	}
	return nil
}

func uninstallComponents(ctx context.Context, kubeClient client.Client, namespace string) error {
	opts := &client.DeleteOptions{}
	selector := client.MatchingLabels{manifestgen.PartOfLabelKey: manifestgen.PartOfLabelValue}
	{
		var list appsv1.DeploymentList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					//logger.Failuref("Deployment/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("Deployment/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list corev1.ServiceList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					//logger.Failuref("Service/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("Service/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list networkingv1.NetworkPolicyList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					//logger.Failuref("NetworkPolicy/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("NetworkPolicy/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list corev1.ServiceAccountList
		if err := kubeClient.List(ctx, &list, client.InNamespace(namespace), selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					//logger.Failuref("ServiceAccount/%s/%s deletion failed: %s", r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("ServiceAccount/%s/%s deleted %s", r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list rbacv1.ClusterRoleList
		if err := kubeClient.List(ctx, &list, selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					//logger.Failuref("ClusterRole/%s deletion failed: %s", r.Name, err.Error())
				} else {
					//logger.Successf("ClusterRole/%s deleted %s", r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list rbacv1.ClusterRoleBindingList
		if err := kubeClient.List(ctx, &list, selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, opts); err != nil {
					//logger.Failuref("ClusterRoleBinding/%s deletion failed: %s", r.Name, err.Error())
				} else {
					//logger.Successf("ClusterRoleBinding/%s deleted %s", r.Name, dryRunStr)
				}
			}
		}
	}
	return nil
}

func uninstallFinalizers(ctx context.Context, kubeClient client.Client) error {
	opts := &client.UpdateOptions{}
	{
		var list sourcev1.GitRepositoryList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					//logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list sourcev1.HelmRepositoryList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					//logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list sourcev1.HelmChartList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					//logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list sourcev1.BucketList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					//logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list kustomizev1.KustomizationList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					//logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list helmv2.HelmReleaseList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					//logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list imagev1.ImagePolicyList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					//logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list imagev1.ImageRepositoryList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					//logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	{
		var list autov1.ImageUpdateAutomationList
		if err := kubeClient.List(ctx, &list, client.InNamespace("")); err == nil {
			for _, r := range list.Items {
				r.Finalizers = []string{}
				if err := kubeClient.Update(ctx, &r, opts); err != nil {
					//logger.Failuref("%s/%s/%s removing finalizers failed: %s", r.Kind, r.Namespace, r.Name, err.Error())
				} else {
					//logger.Successf("%s/%s/%s finalizers deleted %s", r.Kind, r.Namespace, r.Name, dryRunStr)
				}
			}
		}
	}
	return nil
}

func uninstallCustomResourceDefinitions(ctx context.Context, kubeClient client.Client) error {
	selector := client.MatchingLabels{manifestgen.PartOfLabelKey: manifestgen.PartOfLabelValue}
	{
		var list apiextensionsv1.CustomResourceDefinitionList
		if err := kubeClient.List(ctx, &list, selector); err == nil {
			for _, r := range list.Items {
				if err := kubeClient.Delete(ctx, &r, &client.DeleteOptions{}); err != nil {
					//logger.Failuref("CustomResourceDefinition/%s deletion failed: %s", r.Name, err.Error())
				} else {
					//logger.Successf("CustomResourceDefinition/%s deleted %s", r.Name, dryRunStr)
				}
			}
		}
	}
	return nil
}

func uninstallNamespace(ctx context.Context, kubeClient client.Client, namespace string) error {
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if err := kubeClient.Delete(ctx, &ns, &client.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}
