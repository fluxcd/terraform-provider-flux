package validators

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type urlSchemeValidator struct {
	schemes []string
}

func (v urlSchemeValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("url can have scheme %v", v.schemes)
}

func (v urlSchemeValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("url can have scheme %v", v.schemes)
}

func (v urlSchemeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	u, err := url.Parse(req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "could not parse url", err.Error())
		return
	}
	for _, s := range v.schemes {
		if s == u.Scheme {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid URL scheme",
		fmt.Sprintf("Url can have scheme %v", v.schemes),
	)
}

func URLScheme(schemes ...string) validator.String {
	return urlSchemeValidator{schemes: schemes}
}

type mustContainValidator struct {
	contains []string
}

func (v mustContainValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("list must contain %v", v.contains)
}

func (v mustContainValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("list must contain %v", v.contains)
}

func (v mustContainValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	setStr := []string{}
	diag := req.ConfigValue.ElementsAs(ctx, setStr, false)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

OUTER:
	for _, contain := range v.contains {
		for _, str := range setStr {
			if str == contain {
				continue OUTER
			}
		}
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing Required Set Contents",
			fmt.Sprintf("Set has to contain the items %v", v.contains),
		)
		return
	}
}

func MustContain(contains ...string) validator.Set {
	return mustContainValidator{contains: contains}
}
