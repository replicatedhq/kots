package upstream

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	v1_2_0  = "v1.2.0"
	channel = "channel"
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

			replicatedUpstream, err := parseReplicatedURL(u)
			req.NoError(err)
			assert.Equal(t, test.expectedAppSlug, replicatedUpstream.AppSlug)

			if test.expectedVersionLabel != nil || replicatedUpstream.VersionLabel != nil {
				assert.Equal(t, test.expectedVersionLabel, replicatedUpstream.VersionLabel)
			}
		})
	}
}

func Test_releaseToFiles(t *testing.T) {
	tests := []struct {
		name     string
		release  *Release
		expected []UpstreamFile
	}{
		{
			name: "with common prefix",
			release: &Release{
				Manifests: map[string][]byte{
					"manifests/deployment.yaml": []byte("---"),
					"manifests/service.yaml":    []byte("---"),
				},
			},
			expected: []UpstreamFile{
				UpstreamFile{
					Path:    "deployment.yaml",
					Content: []byte("---"),
				},
				UpstreamFile{
					Path:    "service.yaml",
					Content: []byte("---"),
				},
			},
		},
		{
			name: "without common prefix",
			release: &Release{
				Manifests: map[string][]byte{
					"manifests/deployment.yaml": []byte("---"),
					"service.yaml":              []byte("---"),
				},
			},
			expected: []UpstreamFile{
				UpstreamFile{
					Path:    "manifests/deployment.yaml",
					Content: []byte("---"),
				},
				UpstreamFile{
					Path:    "service.yaml",
					Content: []byte("---"),
				},
			},
		},
		{
			name: "common prefix, with userdata",
			release: &Release{
				Manifests: map[string][]byte{
					"manifests/deployment.yaml": []byte("---"),
					"manifests/service.yaml":    []byte("---"),
					"userdata/values.yaml":      []byte("---"),
				},
			},
			expected: []UpstreamFile{
				UpstreamFile{
					Path:    "deployment.yaml",
					Content: []byte("---"),
				},
				UpstreamFile{
					Path:    "service.yaml",
					Content: []byte("---"),
				},
				UpstreamFile{
					Path:    "userdata/values.yaml",
					Content: []byte("---"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := releaseToFiles(test.release)
			req.NoError(err)

			assert.ElementsMatch(t, test.expected, actual)
		})
	}
}
