/*
Copyright 2020 The Flux authors

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
