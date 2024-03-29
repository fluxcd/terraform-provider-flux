// Copyright (c) The Flux authors
// SPDX-License-Identifier: Apache-2.0

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
