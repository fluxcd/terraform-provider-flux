// Copyright (c) The Flux authors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenereateKustomizationYamlWithNoPatches(t *testing.T) {
	result, err := GenerateKustomizationYaml([]string{"foo", "bar"}, []string{})

	expected := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- foo
- bar
`

	assert.Nil(t, err)
	assert.Equal(t, expected, result)
}

func TestGenereateKustomizationYamlWithPatches(t *testing.T) {
	result, err := GenerateKustomizationYaml([]string{"foo", "bar"}, []string{"baz", "buzz"})

	expected := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- foo
- bar
patchesStrategicMerge:
- baz
- buzz
`

	assert.Nil(t, err)
	assert.Equal(t, expected, result)
}

func TestMap(t *testing.T) {
	input := []string{"foo", "bar"}

	expected := []string{"FOO", "BAR"}
	actual := Map(input, strings.ToUpper)

	assert.Equal(t, expected, actual)
}

func TestMapReturnsEmptySliceGivenEmptySlice(t *testing.T) {
	input := []string{}

	expected := []string{}
	actual := Map(input, strings.ToUpper)

	assert.Equal(t, expected, actual)
}

func TestMapReturnsNilSliceGivenNilSlice(t *testing.T) {
	var input []string

	var expected []string
	actual := Map(input, strings.ToUpper)

	assert.Equal(t, expected, actual)
}

func TestGetPatchBases(t *testing.T) {
	input := []string{"foo", "bar"}

	expected := []string{"patch-foo.yaml", "patch-bar.yaml"}
	actual := GetPatchBases(input)

	assert.Equal(t, expected, actual)
}

func TestGenPatchFilePaths(t *testing.T) {
	basePath := "/foo"
	patchNames := []string{"bar"}

	expected := map[string]string{"bar": "/foo/patch-bar.yaml"}
	actual := GenPatchFilePaths(basePath, patchNames)

	assert.Equal(t, expected, actual)
}
