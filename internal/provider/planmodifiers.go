package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

type DefaultModifier struct {
	Default attr.Value
}

func NewDefaultModifier(val attr.Value) DefaultModifier {
	return DefaultModifier{Default: val}
}

func (m DefaultModifier) Description(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to %s", m.Default)
}

func (m DefaultModifier) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to `%s`", m.Default)
}

func (m DefaultModifier) Modify(ctx context.Context, req tfsdk.ModifyAttributePlanRequest, resp *tfsdk.ModifyAttributePlanResponse) {
	/*if !req.AttributePlan.IsNull() {
	    tflog.Debug(ctx, "value is null", map[string]interface{}{"value": req.AttributePlan.String()})
	    return
		}*/
	resp.AttributePlan = m.Default
}
