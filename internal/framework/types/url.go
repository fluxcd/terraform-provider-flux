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
