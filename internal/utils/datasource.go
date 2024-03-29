// Copyright (c) The Flux authors
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bytes"
	"fmt"
	"path"
	"text/template"
)

type KustomizationValues struct {
	Paths   []string
	Patches []string
}

func GenerateKustomizationYaml(paths []string, patches []string) (string, error) {
	var t *template.Template
	var err error

	if len(patches) == 0 {
		t, err = template.New("kustomize").Parse(kustomizeTemplateString)
	} else {
		t, err = template.New("kustomize").Parse(kustomizeWithPatchesTemplateString)
	}

	if err != nil {
		return "", err
	}

	var kustomize bytes.Buffer
	values := KustomizationValues{paths, patches}
	err = t.Execute(&kustomize, values)
	if err != nil {
		return "", err
	}

	return kustomize.String(), nil
}

const kustomizeTemplateString = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
{{- range .Paths }}
- {{.}}
{{- end }}
`

const kustomizeWithPatchesTemplateString = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
{{- range .Paths }}
- {{.}}
{{- end }}
patchesStrategicMerge:
{{- range .Patches }}
- {{.}}
{{- end }}
`

func Map(vs []string, f func(string) string) []string {
	if vs == nil {
		return nil
	}
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

func GetPatchBases(patchNames []string) []string {
	f := func(s string) string {
		return fmt.Sprintf("patch-%s.yaml", s)
	}

	return Map(patchNames, f)
}

func GenPatchFilePaths(basePath string, patchNames []string) map[string]string {
	f := func(filename string) string {
		return path.Join(basePath, filename)
	}

	patchBases := GetPatchBases(patchNames)
	filePaths := Map(patchBases, f)

	patchMap := make(map[string]string)
	for i, v := range filePaths {
		key := patchNames[i]
		patchMap[key] = v
	}

	return patchMap
}
