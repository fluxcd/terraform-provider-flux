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

package provider

import (
	"testing"

	"github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/stretchr/testify/require"
)

func TestGetClusterDomainFromEventsAddress(t *testing.T) {
	defaultClusterDomain := install.MakeDefaultOptions().ClusterDomain

	tests := []struct {
		name                  string
		eventsAddress         string
		namespace             string
		expectedClusterDomain string
		expectedInferred      bool
		expectedError         string
	}{
		{
			name:                  "infers cluster domain from fully qualified service address",
			eventsAddress:         "http://notification-controller.flux-system.svc.cluster.local./",
			namespace:             "flux-system",
			expectedClusterDomain: "cluster.local",
			expectedInferred:      true,
		},
		{
			name:                  "infers custom cluster domain",
			eventsAddress:         "http://notification-controller.flux-system.svc.corp.example.internal./",
			namespace:             "flux-system",
			expectedClusterDomain: "corp.example.internal",
			expectedInferred:      true,
		},
		{
			name:                  "falls back to default for service host without cluster domain",
			eventsAddress:         "http://notification-controller.flux-system.svc/",
			namespace:             "flux-system",
			expectedClusterDomain: defaultClusterDomain,
			expectedInferred:      false,
		},
		{
			name:                  "falls back to default for short service host",
			eventsAddress:         "http://notification-controller/",
			namespace:             "flux-system",
			expectedClusterDomain: defaultClusterDomain,
			expectedInferred:      false,
		},
		{
			name:                  "falls back to default for unexpected host shape",
			eventsAddress:         "http://other-service.flux-system.svc.cluster.local./",
			namespace:             "flux-system",
			expectedClusterDomain: defaultClusterDomain,
			expectedInferred:      false,
		},
		{
			name:          "fails for hostless address",
			eventsAddress: "notification-controller",
			namespace:     "flux-system",
			expectedError: "events address does not contain a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusterDomain, inferred, err := getClusterDomainFromEventsAddress(tt.eventsAddress, tt.namespace)
			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedClusterDomain, clusterDomain)
			require.Equal(t, tt.expectedInferred, inferred)
		})
	}
}
