package replicatedapp

import (
	"net/url"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	v1_2_0 = "v1.2.0"
)

func Test_parseReplicatedURL(t *testing.T) {
	tests := []struct {
		name                 string
		uri                  string
		expectedAppSlug      string
		expectedChannel      *string
		expectedVersionLabel *string
		expectedSequence     *int
	}{
		{
			name:                 "replicated://app-slug",
			uri:                  "replicated://app-slug",
			expectedAppSlug:      "app-slug",
			expectedChannel:      nil,
			expectedVersionLabel: nil,
			expectedSequence:     nil,
		},
		{
			name:                 "replicated://app-slug@v1.2.0",
			uri:                  "replicated://app-slug@v1.2.0",
			expectedAppSlug:      "app-slug",
			expectedChannel:      nil,
			expectedVersionLabel: &v1_2_0,
			expectedSequence:     nil,
		},
		{
			name:                 "replicated://app-slug/channel",
			uri:                  "replicated://app-slug/channel",
			expectedAppSlug:      "app-slug",
			expectedChannel:      &channel,
			expectedVersionLabel: nil,
			expectedSequence:     nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			u, err := url.ParseRequestURI(test.uri)
			req.NoError(err)

			replicatedUpstream, err := ParseReplicatedURL(u)
			req.NoError(err)
			assert.Equal(t, test.expectedAppSlug, replicatedUpstream.AppSlug)

			if test.expectedVersionLabel != nil || replicatedUpstream.VersionLabel != nil {
				assert.Equal(t, test.expectedVersionLabel, replicatedUpstream.VersionLabel)
			}
		})
	}
}

func Test_getReplicatedAppEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		license     *kotsv1beta1.License
		isEmbedded  bool
		envEndpoint string
		want        string
		wantError   bool
	}{
		{
			name: "kots install with full endpoint",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Endpoint: "https://replicated.app",
				},
			},
			isEmbedded: false,
			want:       "https://replicated.app",
		},
		{
			name: "kots install with endpoint including port",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Endpoint: "https://replicated.app:8443",
				},
			},
			isEmbedded: false,
			want:       "https://replicated.app:8443",
		},
		{
			name:        "embedded cluster with env endpoint",
			license:     &kotsv1beta1.License{},
			isEmbedded:  true,
			envEndpoint: "https://replicated.app",
			want:        "https://replicated.app",
		},
		{
			name:       "embedded cluster without env endpoint",
			license:    &kotsv1beta1.License{},
			isEmbedded: true,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// Setup environment
			if tt.isEmbedded {
				t.Setenv("EMBEDDED_CLUSTER_ID", "123")

				if tt.envEndpoint != "" {
					t.Setenv("REPLICATED_API_ENDPOINT", tt.envEndpoint)
				}
			}

			result, err := getReplicatedAppEndpoint(tt.license)

			if tt.wantError {
				req.Error(err)
				return
			}

			req.NoError(err)
			assert.Equal(t, tt.want, result)
		})
	}
}
