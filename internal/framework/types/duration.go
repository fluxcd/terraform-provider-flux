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
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type DurationType struct {
	basetypes.StringType
}

func (t DurationType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	val, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	strVal, ok := val.(types.String)
	if !ok {
		return nil, fmt.Errorf("value of unexpected type")
	}
	if strVal.ValueString() == "" {
		return DurationNull(), nil
	}
	d, err := time.ParseDuration(strVal.ValueString())
	if err != nil {
		return nil, fmt.Errorf("could not parse duration: %w", err)
	}
	return Duration{
		StringValue: strVal,
		duration:    d,
	}, nil
}

type Duration struct {
	basetypes.StringValue
	duration time.Duration
}

func (v Duration) ValueDuration() time.Duration {
	return v.duration
}

func DurationNull() Duration {
	return Duration{
		StringValue: types.StringNull(),
	}
}

func DurationUnknown() Duration {
	return Duration{
		StringValue: types.StringUnknown(),
	}
}

func DurationValue(value time.Duration) Duration {
	return Duration{
		StringValue: types.StringValue(value.String()),
		duration:    value,
	}
}
