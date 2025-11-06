package replicatedapp

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
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
		licenseWrapper := licensewrapper.LicenseWrapper{V1: license}
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
		request, err := r.GetRequest("GET", &licenseWrapper, cursor.Cursor, channel)
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
			licenseWrapper := licensewrapper.LicenseWrapper{V1: test.license}
			url, err := makeLicenseURL(&licenseWrapper, test.selectedChannelID)
			require.NoError(t, err)
			assert.Equal(t, test.expectedURL, url)
		})
	}
}

func Test_getLicenseFromAPI_HeaderSet(t *testing.T) {
	tests := []struct {
		name                  string
		license               *licensewrapper.LicenseWrapper
		expectedHeaderVersion string // empty string means header should not be set
	}{
		{
			name: "v1beta1 license does not set header",
			license: &licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID: "test-license-id",
						AppSlug:   "test-app",
					},
				},
			},
			expectedHeaderVersion: "", // no header for v1beta1
		},
		{
			name: "v1beta2 license sets v1beta2 header",
			license: &licensewrapper.LicenseWrapper{
				V2: &kotsv1beta2.License{
					Spec: kotsv1beta2.LicenseSpec{
						LicenseID: "test-license-id-v2",
						AppSlug:   "test-app-v2",
					},
				},
			},
			expectedHeaderVersion: "v1beta2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a test server that captures the request headers
			var capturedHeaders http.Header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedHeaders = r.Header.Clone()

				// Verify basic auth is set correctly
				licenseID := test.license.GetLicenseID()
				expectedAuth := fmt.Sprintf("Basic %s",
					base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", licenseID, licenseID))))
				assert.Equal(t, expectedAuth, r.Header.Get("Authorization"))

				// Return a valid license response based on the license type
				var response string
				if test.license.IsV1() {
					response = fmt.Sprintf(`apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: test-license
spec:
  licenseID: %s
  appSlug: %s
`, licenseID, test.license.GetAppSlug())
				} else {
					response = fmt.Sprintf(`apiVersion: kots.io/v1beta2
kind: License
metadata:
  name: test-license
spec:
  licenseID: %s
  appSlug: %s
`, licenseID, test.license.GetAppSlug())
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(response))
			}))
			defer server.Close()

			// Call the function
			licenseData, err := getLicenseFromAPI(server.URL, test.license)
			require.NoError(t, err)
			require.NotNil(t, licenseData)

			// Verify the X-Replicated-License-Version header
			if test.expectedHeaderVersion == "" {
				// Header should NOT be set for v1beta1
				assert.Empty(t, capturedHeaders.Get("X-Replicated-License-Version"),
					"X-Replicated-License-Version header should not be set for v1beta1")
			} else {
				// Header should be set for v1beta2
				assert.Equal(t, test.expectedHeaderVersion, capturedHeaders.Get("X-Replicated-License-Version"),
					"X-Replicated-License-Version header should be set to %s", test.expectedHeaderVersion)
			}

			// Verify the returned license matches the expected version
			if test.license.IsV1() {
				assert.True(t, licenseData.License.IsV1())
				assert.False(t, licenseData.License.IsV2())
			} else {
				assert.False(t, licenseData.License.IsV1())
				assert.True(t, licenseData.License.IsV2())
			}
		})
	}
}
