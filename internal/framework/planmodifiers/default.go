package planmodifiers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type stringDefaultModifier struct {
	value string
}

func (m stringDefaultModifier) Description(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to %v", m.value)
}

func (m stringDefaultModifier) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to `%v`", m.value)
}

func (m stringDefaultModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if !req.PlanValue.IsUnknown() && !req.PlanValue.IsNull() {
		return
	}
	resp.PlanValue = types.StringValue(m.value)
}

func DefaultStringValue(value string) planmodifier.String {
	return &stringDefaultModifier{value: value}
}

type boolDefaultModifier struct {
	value bool
}

func (m boolDefaultModifier) Description(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to %v", m.value)
}

func (m boolDefaultModifier) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to `%v`", m.value)
}

func (m boolDefaultModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if !req.PlanValue.IsUnknown() && !req.PlanValue.IsNull() {
		return
	}
	resp.PlanValue = types.BoolValue(m.value)
}

func DefaultBoolValue(value bool) planmodifier.Bool {
	return boolDefaultModifier{value: value}
}

type stringSetDefaultModifier struct {
	value []string
}

func (m stringSetDefaultModifier) Description(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to %v", m.value)
}

func (m stringSetDefaultModifier) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to `%v`", m.value)
}

func (m stringSetDefaultModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if !req.PlanValue.IsUnknown() && !req.PlanValue.IsNull() {
		return
	}
	attrValues := []attr.Value{}
	for _, v := range m.value {
		attrValues = append(attrValues, types.StringValue(v))
	}
	set := types.SetValueMust(types.StringType, attrValues)
	resp.PlanValue = set
}

func DefaultStringSetValue(value []string) planmodifier.Set {
	return stringSetDefaultModifier{value: value}
}
