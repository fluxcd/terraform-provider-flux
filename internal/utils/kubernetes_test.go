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

package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestGetContainers(t *testing.T) {
	containers := []corev1.Container{
		{
			Name: "foo",
		},
		{
			Name: "foo",
		},
		{
			Name: "baz",
		},
		{
			Name: "manager",
		},
		{
			Name: "test",
		},
	}
	tests := []struct {
		name          string
		containers    []corev1.Container
		containerName string
		expectedError error
	}{
		{
			name:          "container found",
			containers:    containers,
			containerName: "manager",
		},
		{
			name:          "container not foundfound",
			containers:    containers,
			containerName: "flux",
			expectedError: fmt.Errorf("could not find container: flux"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := GetContainer(tt.containers, tt.containerName)
			if tt.expectedError == nil {
				require.NoError(t, err)
				require.Equal(t, tt.containerName, c.Name)
				return
			}
			require.ErrorContains(t, err, tt.expectedError.Error())
		})
	}
}

func TestGetArgValue(t *testing.T) {
	args := []string{
		"--foo=bar",
		"--test",
		"value",
		"--hello=world",
	}
	tests := []struct {
		name          string
		args          []string
		arg           string
		expectedValue string
		expectedError error
	}{
		{
			name:          "arg with equal",
			args:          args,
			arg:           "--foo",
			expectedValue: "bar",
		},
		{
			name:          "arg with separate item",
			args:          args,
			arg:           "--test",
			expectedValue: "value",
		},
		{
			name:          "not exists",
			args:          args,
			arg:           "--baz",
			expectedError: fmt.Errorf("arg with name not found: --baz"),
		},
		{
			name:          "empty arg name",
			args:          args,
			arg:           "",
			expectedError: fmt.Errorf("arg name cannot be empty"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := corev1.Container{
				Args: tt.args,
			}
			value, err := GetArgValue(c, tt.arg)
			if tt.expectedError == nil {
				require.NoError(t, err)
				require.Equal(t, tt.expectedValue, value)
				return
			}
			require.ErrorContains(t, err, tt.expectedError.Error())
		})
	}
}
