package replicatedapp

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	channel = "channel"
)

func Test_getRequest(t *testing.T) {
	beta := "beta"
	unstable := "unstable"
	version := "1.1.0"
	tests := []struct {
		endpoint        string
		appSlug         string
		channel         *string
		channelSequence string
		versionLabel    *string
		isEC            bool
		expectedURL     string
	}{
		{
			endpoint:        "https://replicated-app",
			appSlug:         "sluggy1",
			channel:         nil,
			channelSequence: "",
			versionLabel:    nil,
			expectedURL:     "https://replicated-app/release/sluggy1?channelSequence=&isEmbeddedCluster=false&isSemverSupported=true&licenseSequence=23&selectedChannelId=channel",
		},
		{
			endpoint:        "http://localhost:30016",
			appSlug:         "sluggy2",
			channel:         &beta,
			channelSequence: "",
			versionLabel:    nil,
			expectedURL:     "http://localhost:30016/release/sluggy2/beta?channelSequence=&isEmbeddedCluster=false&isSemverSupported=true&licenseSequence=23&selectedChannelId=channel",
		},
		{
			endpoint:        "https://replicated-app",
			appSlug:         "sluggy3",
			channel:         &unstable,
			channelSequence: "10",
			versionLabel:    nil,
			expectedURL:     "https://replicated-app/release/sluggy3/unstable?channelSequence=10&isEmbeddedCluster=false&isSemverSupported=true&licenseSequence=23&selectedChannelId=channel",
		},
		{
			endpoint:        "https://replicated-app",
			appSlug:         "sluggy3",
			channel:         &unstable,
			channelSequence: "",
			versionLabel:    &version,
			expectedURL:     "https://replicated-app/release/sluggy3/unstable?channelSequence=&isEmbeddedCluster=false&isSemverSupported=true&licenseSequence=23&selectedChannelId=channel&versionLabel=1.1.0",
		},
		{
			endpoint:        "https://replicated-app",
			appSlug:         "sluggy3",
			channel:         &unstable,
			channelSequence: "",
			versionLabel:    &version,
			isEC:            true,
			expectedURL:     "https://replicated-app/release/sluggy3/unstable?channelSequence=&isEmbeddedCluster=true&isSemverSupported=true&licenseSequence=23&selectedChannelId=channel&versionLabel=1.1.0",
		},
	}

	req := require.New(t)
	for _, test := range tests {
		license := &kotsv1beta1.License{
			Spec: kotsv1beta1.LicenseSpec{
				Endpoint:        test.endpoint,
				AppSlug:         test.appSlug,
				LicenseSequence: 23,
				ChannelID:       "channel",
			},
		}
		r := &ReplicatedUpstream{
			Channel:      test.channel,
			VersionLabel: test.versionLabel,
		}
		cursor := ReplicatedCursor{
			Cursor: test.channelSequence,
		}
		if test.channel != nil {
			cursor.ChannelName = *test.channel
		}
		if test.isEC {
			t.Setenv("EMBEDDED_CLUSTER_ID", "123")
			t.Setenv("REPLICATED_APP_ENDPOINT", "https://replicated-app")
		}
		request, err := r.GetRequest("GET", license, cursor.Cursor, channel)
		req.NoError(err)
		assert.Equal(t, test.expectedURL, request.URL.String())
	}
}

func Test_makeLicenseURL(t *testing.T) {
	tests := []struct {
		description       string
		license           *kotsv1beta1.License
		selectedChannelID string
		expectedURL       string
	}{
		{
			description: "with channel",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Endpoint:        "https://replicated-app",
					AppSlug:         "slug1",
					LicenseSequence: 42,
				},
			},
			selectedChannelID: "channel1",
			expectedURL:       "https://replicated-app/license/slug1?licenseSequence=42&selectedChannelId=channel1",
		},
		{
			description: "without channel",
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Endpoint:        "https://replicated-app",
					AppSlug:         "slug2",
					LicenseSequence: 42,
				},
			},
			selectedChannelID: "",
			expectedURL:       "https://replicated-app/license/slug2?licenseSequence=42",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			url, err := makeLicenseURL(test.license, test.selectedChannelID)
			require.NoError(t, err)
			assert.Equal(t, test.expectedURL, url)
		})
	}
}
