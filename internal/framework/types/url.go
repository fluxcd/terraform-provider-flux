package types

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type URLType struct {
	basetypes.StringType
}

func (t URLType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	val, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	strVal, ok := val.(types.String)
	if !ok {
		return nil, fmt.Errorf("value of unexpected type")
	}
	u, err := url.Parse(strVal.ValueString())
	if err != nil {
		return nil, err
	}
	return URL{
		StringValue: strVal,
		url:         u,
	}, nil
}

type URL struct {
	basetypes.StringValue
	url *url.URL
}

func (v URL) ValueURL() *url.URL {
	return v.url
}

func URLNull() URL {
	return URL{
		StringValue: types.StringNull(),
	}
}

func URLUnknown() URL {
	return URL{
		StringValue: types.StringUnknown(),
	}
}

func URLValue(value *url.URL) URL {
	return URL{
		StringValue: types.StringValue(value.String()),
		url:         value,
	}
}
