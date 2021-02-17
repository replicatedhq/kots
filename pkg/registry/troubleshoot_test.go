package registry

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_UpdateCollectorSpecsWithRegistryData(t *testing.T) {
	tests := []struct {
		name               string
		knownImages        []kotsv1beta1.InstallationImage
		localRegistryInfo  *registrytypes.RegistrySettings
		license            *kotsv1beta1.License
		collectors         []*troubleshootv1beta2.Collect
		expectedCollectors []*troubleshootv1beta2.Collect
	}{
		{
			name:               "empty spec, no change",
			knownImages:        nil,
			localRegistryInfo:  nil,
			license:            nil,
			collectors:         []*troubleshootv1beta2.Collect{},
			expectedCollectors: []*troubleshootv1beta2.Collect{},
		},
		{
			name:              "valid spec, no change",
			knownImages:       nil,
			localRegistryInfo: nil,
			license:           nil,
			collectors: []*troubleshootv1beta2.Collect{
				{
					ClusterInfo: &troubleshootv1beta2.ClusterInfo{},
				},
			},
			expectedCollectors: []*troubleshootv1beta2.Collect{
				{
					ClusterInfo: &troubleshootv1beta2.ClusterInfo{},
				},
			},
		},
		{
			name: "run collector, public image",
			knownImages: []kotsv1beta1.InstallationImage{
				{
					Image:     "docker.io/bitnami/postgres:11",
					IsPrivate: false,
				},
			},
			localRegistryInfo: nil,
			license:           nil,
			collectors: []*troubleshootv1beta2.Collect{
				{
					Run: &troubleshootv1beta2.Run{
						Image:           "docker.io/bitnami/postgres:11",
						ImagePullSecret: nil,
					},
				},
			},
			expectedCollectors: []*troubleshootv1beta2.Collect{
				{
					Run: &troubleshootv1beta2.Run{
						Image:           "docker.io/bitnami/postgres:11",
						ImagePullSecret: nil,
					},
				},
			},
		},
		{
			name:        "run collector, public image, private local registry",
			knownImages: []kotsv1beta1.InstallationImage{},
			localRegistryInfo: &registrytypes.RegistrySettings{
				Hostname:  "ttl.sh",
				Namespace: "abc",
				Username:  "user",
				Password:  "pass",
			},
			license: nil,
			collectors: []*troubleshootv1beta2.Collect{
				{
					Run: &troubleshootv1beta2.Run{
						Image:           "docker.io/bitnami/postgres:11",
						ImagePullSecret: nil,
					},
				},
			},
			expectedCollectors: []*troubleshootv1beta2.Collect{
				{
					Run: &troubleshootv1beta2.Run{
						Image: "ttl.sh/abc/postgres:11",
						ImagePullSecret: &troubleshootv1beta2.ImagePullSecrets{
							SecretType: "kubernetes.io/dockerconfigjson",
							Data: map[string]string{
								".dockerconfigjson": "eyJhdXRocyI6eyJ0dGwuc2giOnsiYXV0aCI6ImRYTmxjanB3WVhOeiJ9fX0=",
							},
						},
					},
				},
			},
		},
		{
			name: "run collector, replicated registry image, no private local registry",
			knownImages: []kotsv1beta1.InstallationImage{
				{
					Image:     "registry.replicated.com/my-app/my-image:abcdef",
					IsPrivate: true,
				},
			},
			localRegistryInfo: nil,
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					LicenseID: "licenseid",
					AppSlug:   "app-slug",
				},
			},
			collectors: []*troubleshootv1beta2.Collect{
				{
					Run: &troubleshootv1beta2.Run{
						Image: "registry.replicated.com/my-app/my-image:abcdef",
					},
				},
			},
			expectedCollectors: []*troubleshootv1beta2.Collect{
				{
					Run: &troubleshootv1beta2.Run{
						Image: "registry.replicated.com/my-app/my-image:abcdef",
						ImagePullSecret: &troubleshootv1beta2.ImagePullSecrets{
							SecretType: "kubernetes.io/dockerconfigjson",
							Data: map[string]string{
								".dockerconfigjson": "eyJhdXRocyI6eyJyZWdpc3RyeS5yZXBsaWNhdGVkLmNvbSI6eyJhdXRoIjoiYkdsalpXNXpaV2xrT214cFkyVnVjMlZwWkE9PSJ9fX0=",
							},
						},
					},
				},
			},
		},
		{
			name: "run collector, private image (not replicated), no private local registry",
			knownImages: []kotsv1beta1.InstallationImage{
				{
					Image:     "quay.io/my-app/my-image:abcdef",
					IsPrivate: true,
				},
			},
			localRegistryInfo: nil,
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					LicenseID: "licenseid",
					AppSlug:   "app-slug",
				},
			},
			collectors: []*troubleshootv1beta2.Collect{
				{
					Run: &troubleshootv1beta2.Run{
						Image: "quay.io/my-app/my-image:abcdef",
					},
				},
			},
			expectedCollectors: []*troubleshootv1beta2.Collect{
				{
					Run: &troubleshootv1beta2.Run{
						Image: "proxy.replicated.com/proxy/app-slug/quay.io/my-app/my-image:abcdef",
						ImagePullSecret: &troubleshootv1beta2.ImagePullSecrets{
							SecretType: "kubernetes.io/dockerconfigjson",
							Data: map[string]string{
								".dockerconfigjson": "eyJhdXRocyI6eyJwcm94eS5yZXBsaWNhdGVkLmNvbSI6eyJhdXRoIjoiYkdsalpXNXpaV2xrT214cFkyVnVjMlZwWkE9PSJ9fX0=",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actualCollectors, err := UpdateCollectorSpecsWithRegistryData(test.collectors, test.localRegistryInfo, test.knownImages, test.license)
			req.NoError(err)

			assert.Equal(t, test.expectedCollectors, actualCollectors)
		})
	}
}
