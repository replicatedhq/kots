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
		expectedURL     string
	}{
		{
			endpoint:        "https://replicated-app",
			appSlug:         "sluggy1",
			channel:         nil,
			channelSequence: "",
			versionLabel:    nil,
			expectedURL:     "https://replicated-app/release/sluggy1?channelSequence=&isSemverSupported=true&licenseSequence=23",
		},
		{
			endpoint:        "http://localhost:30016",
			appSlug:         "sluggy2",
			channel:         &beta,
			channelSequence: "",
			versionLabel:    nil,
			expectedURL:     "http://localhost:30016/release/sluggy2/beta?channelSequence=&isSemverSupported=true&licenseSequence=23",
		},
		{
			endpoint:        "https://replicated-app",
			appSlug:         "sluggy3",
			channel:         &unstable,
			channelSequence: "10",
			versionLabel:    nil,
			expectedURL:     "https://replicated-app/release/sluggy3/unstable?channelSequence=10&isSemverSupported=true&licenseSequence=23",
		},
		{
			endpoint:        "https://replicated-app",
			appSlug:         "sluggy3",
			channel:         &unstable,
			channelSequence: "",
			versionLabel:    &version,
			expectedURL:     "https://replicated-app/release/sluggy3/unstable?channelSequence=&isSemverSupported=true&licenseSequence=23&versionLabel=1.1.0",
		},
	}

	req := require.New(t)
	for _, test := range tests {
		license := &kotsv1beta1.License{
			Spec: kotsv1beta1.LicenseSpec{
				Endpoint:        test.endpoint,
				AppSlug:         test.appSlug,
				LicenseSequence: 23,
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
		request, err := r.GetRequest("GET", license, cursor)
		req.NoError(err)
		assert.Equal(t, test.expectedURL, request.URL.String())
	}
}
