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
		expectedHost         string
		expectedAppSlug      string
		expectedChannel      *string
		expectedVersionLabel *string
		expectedSequence     *int
	}{
		{
			name:                 "replicated://app-slug",
			uri:                  "replicated://app-slug",
			expectedHost:         "replicated.app",
			expectedAppSlug:      "app-slug",
			expectedChannel:      nil,
			expectedVersionLabel: nil,
			expectedSequence:     nil,
		},
		{
			name:                 "replicated://app-slug@v1.2.0",
			uri:                  "replicated://app-slug@v1.2.0",
			expectedHost:         "replicated.app",
			expectedAppSlug:      "app-slug",
			expectedChannel:      nil,
			expectedVersionLabel: &v1_2_0,
			expectedSequence:     nil,
		},
		{
			name:                 "replicated://app-slug/channel",
			uri:                  "replicated://app-slug/channel",
			expectedHost:         "replicated.app",
			expectedAppSlug:      "app-slug",
			expectedChannel:      &channel,
			expectedVersionLabel: nil,
			expectedSequence:     nil,
		},
		{
			name:                 "replicated://app-slug?host=my.hostname",
			uri:                  "replicated://app-slug?host=my.hostname",
			expectedHost:         "my.hostname",
			expectedAppSlug:      "app-slug",
			expectedChannel:      nil,
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
			assert.Equal(t, test.expectedHost, replicatedUpstream.Host)
			assert.Equal(t, test.expectedAppSlug, replicatedUpstream.AppSlug)

			if test.expectedVersionLabel != nil || replicatedUpstream.VersionLabel != nil {
				assert.Equal(t, test.expectedVersionLabel, replicatedUpstream.VersionLabel)
			}
		})
	}
}
