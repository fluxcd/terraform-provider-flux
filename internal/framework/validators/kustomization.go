package validators

import (
	"context"
	// "fmt"

	// "github.com/fluxcd/flux2/pkg/manifestgen/kustomization"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

type kustomizationOverrideValidator struct {
}

func (v kustomizationOverrideValidator) Description(ctx context.Context) string {
	return "Kustomization override contains required resources."
}

func (v kustomizationOverrideValidator) MarkdownDescription(ctx context.Context) string {
	return "Kustomization override contains required resources."
}

func (v kustomizationOverrideValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	kus := &kustypes.Kustomization{}
	err := yaml.Unmarshal([]byte(req.ConfigValue.ValueString()), kus)
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "could not parse kustomization", err.Error())
		return
	}
	if !contains(kus.Resources, "gotk-components.yaml") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing required resource.",
			"Kustomization resource must contain: gotk-components.yaml",
		)
	}
	if !contains(kus.Resources, "gotk-sync.yaml") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing required resource.",
			"Kustomization resource must contain: gotk-sync.yaml",
		)
	}
}

func KustomizationOverride() validator.String {
	return kustomizationOverrideValidator{}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
