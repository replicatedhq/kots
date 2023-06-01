package registry

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/registry/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_UpdateCollectorSpecsWithRegistryData(t *testing.T) {
	tests := []struct {
		name               string
		installation       kotsv1beta1.Installation
		localRegistryInfo  registrytypes.RegistrySettings
		license            *kotsv1beta1.License
		collectors         []*troubleshootv1beta2.Collect
		expectedCollectors []*troubleshootv1beta2.Collect
	}{
		{
			name:               "empty spec, no change",
			installation:       kotsv1beta1.Installation{},
			localRegistryInfo:  registrytypes.RegistrySettings{},
			license:            nil,
			collectors:         []*troubleshootv1beta2.Collect{},
			expectedCollectors: []*troubleshootv1beta2.Collect{},
		},
		{
			name:              "valid spec, no change",
			installation:      kotsv1beta1.Installation{},
			localRegistryInfo: registrytypes.RegistrySettings{},
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
			installation: kotsv1beta1.Installation{
				Spec: kotsv1beta1.InstallationSpec{
					KnownImages: []kotsv1beta1.InstallationImage{
						{
							Image:     "docker.io/bitnami/postgres:11",
							IsPrivate: false,
						},
					},
				},
			},
			localRegistryInfo: registrytypes.RegistrySettings{},
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
			name:         "run collector, public image, private local registry",
			installation: kotsv1beta1.Installation{},
			localRegistryInfo: registrytypes.RegistrySettings{
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
			installation: kotsv1beta1.Installation{
				Spec: kotsv1beta1.InstallationSpec{
					KnownImages: []kotsv1beta1.InstallationImage{
						{
							Image:     "registry.replicated.com/my-app/my-image:abcdef",
							IsPrivate: true,
						},
					},
				},
			},
			localRegistryInfo: registrytypes.RegistrySettings{},
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
								".dockerconfigjson": "eyJhdXRocyI6eyJwcm94eS5yZXBsaWNhdGVkLmNvbSI6eyJhdXRoIjoiYkdsalpXNXpaV2xrT214cFkyVnVjMlZwWkE9PSJ9LCJyZWdpc3RyeS5yZXBsaWNhdGVkLmNvbSI6eyJhdXRoIjoiYkdsalpXNXpaV2xrT214cFkyVnVjMlZwWkE9PSJ9fX0=",
							},
						},
					},
				},
			},
		},
		{
			name: "run collector, private image (not replicated), no private local registry",
			installation: kotsv1beta1.Installation{
				Spec: kotsv1beta1.InstallationSpec{
					KnownImages: []kotsv1beta1.InstallationImage{
						{
							Image:     "quay.io/my-app/my-image:abcdef",
							IsPrivate: true,
						},
					},
				},
			},
			localRegistryInfo: types.RegistrySettings{},
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
								".dockerconfigjson": "eyJhdXRocyI6eyJwcm94eS5yZXBsaWNhdGVkLmNvbSI6eyJhdXRoIjoiYkdsalpXNXpaV2xrT214cFkyVnVjMlZwWkE9PSJ9LCJyZWdpc3RyeS5yZXBsaWNhdGVkLmNvbSI6eyJhdXRoIjoiYkdsalpXNXpaV2xrT214cFkyVnVjMlZwWkE9PSJ9fX0=",
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

			actualCollectors, err := UpdateCollectorSpecsWithRegistryData(test.collectors, test.localRegistryInfo, test.installation, test.license, nil)
			req.NoError(err)

			assert.Equal(t, test.expectedCollectors, actualCollectors)
		})
	}
}

func Test_rewriteImage(t *testing.T) {
	tests := []struct {
		name    string
		image   string
		newHost string
		newNS   string
		want    string
	}{
		{
			name:    "rewrite with namespace",
			image:   "quay.io/replicatedhq/image:alpine-3.5",
			newHost: "localhost:30000",
			newNS:   "ns",
			want:    "localhost:30000/ns/image:alpine-3.5",
		},
		{
			name:    "rewrite without namespace",
			image:   "quay.io/replicatedhq/image:alpine-3.5",
			newHost: "localhost:30000",
			newNS:   "",
			want:    "localhost:30000/image:alpine-3.5",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := rewriteImage(test.newHost, test.newNS, test.image)
			assert.Equal(t, test.want, got)
		})
	}
}
