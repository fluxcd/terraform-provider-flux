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

package validators

import (
	"context"

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
