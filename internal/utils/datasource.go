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
