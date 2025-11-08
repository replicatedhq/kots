package pull

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Pull(t *testing.T) {
	data := `apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: expiredtestlicense
spec:
  licenseID: VJDJXAPDAStK62ijnUnIC1zJOW0A2t7z
  licenseType: trial
  customerName: ExpiredTestLicense
  appSlug: testkotsapp
  channelName: Unstable
  licenseSequence: 1
  endpoint: 'http://replicated-app:3000'
  entitlements:
    expires_at:
      title: Expiration
      description: License Expiration
      value: '2019-02-06T08:00:00Z'
      valueType: String
  signature: >-
    eyJsaWNlbnNlRGF0YSI6ImV5SmhjR2xXWlhKemFXOXVJam9pYTI5MGN5NXBieTkyTVdKbGRHRXhJaXdpYTJsdVpDSTZJa3hwWTJWdWMyVWlMQ0p0WlhSaFpHRjBZU0k2ZXlKdVlXMWxJam9pWlhod2FYSmxaSFJsYzNSc2FXTmxibk5sSW4wc0luTndaV01pT25zaWJHbGpaVzV6WlVsRUlqb2lWa3BFU2xoQlVFUkJVM1JMTmpKcGFtNVZia2xETVhwS1QxY3dRVEowTjNvaUxDSnNhV05sYm5ObFZIbHdaU0k2SW5SeWFXRnNJaXdpWTNWemRHOXRaWEpPWVcxbElqb2lSWGh3YVhKbFpGUmxjM1JNYVdObGJuTmxJaXdpWVhCd1UyeDFaeUk2SW5SbGMzUnJiM1J6WVhCd0lpd2lZMmhoYm01bGJFNWhiV1VpT2lKVmJuTjBZV0pzWlNJc0lteHBZMlZ1YzJWVFpYRjFaVzVqWlNJNk1Td2laVzVrY0c5cGJuUWlPaUpvZEhSd09pOHZjbVZ3YkdsallYUmxaQzFoY0hBNk16QXdNQ0lzSW1WdWRHbDBiR1Z0Wlc1MGN5STZleUpsZUhCcGNtVnpYMkYwSWpwN0luUnBkR3hsSWpvaVJYaHdhWEpoZEdsdmJpSXNJbVJsYzJOeWFYQjBhVzl1SWpvaVRHbGpaVzV6WlNCRmVIQnBjbUYwYVc5dUlpd2lkbUZzZFdVaU9pSXlNREU1TFRBeUxUQTJWREE0T2pBd09qQXdXaUlzSW5aaGJIVmxWSGx3WlNJNklsTjBjbWx1WnlKOWZYMTkiLCJpbm5lclNpZ25hdHVyZSI6ImV5SnNhV05sYm5ObFUybG5ibUYwZFhKbElqb2lXRU16TDFwUk1XWmlhMlp2WlRkNWNEQjRZemhLYjFseFREQm1PRkZTY2tkeUsweHlRVFpyTDJwdWVGZGlPVE56ZWxoQlNrWldlVWh1ZWpSamVWRlJNRGRtVDFjemFIaE1SbXhPZW1rd1pHcDJUa2RxV1hocVpFNTZaMkV5U1VzdmJEQjZla2d5Um1GeFNFRllLM056VkRSa2FFSklURlY2TmxGVU9IaGpka1JsZDNNM1ZYYzRlV0pqYVdOalFVSnJXUzk2V201M1pXRk5aSGxCYTJaRFVWUnJkSFY2T0hOak5rWXZZbWxXYVhGeGNuSmlOamhuVUdnMldFNU9SRk56YTBSeVptWkhXREJTTTFsTlFYTldkbGw0V2tOWFNtWlZhSGM0TjBaQlZWaFpVbWh4V21SV2IzaFFkRkZtWm1ocVdEaFZNVW8xYUVaalNtTlBjVGQ2VWxZME9VWXdNV2t2VUhodVV6UjNkVE5MUjFkYWNVWjZhbFJNWlVsSU9UaFVWeXRPYkhObmFYZ3hXWE5qV0dZelNtNUtZVzg1V0hCRU5XUnNha3N4ZEZsUGJITllSR0pyTTNjd2VrOTFOamhEVUZORE1ESkJQVDBpTENKd2RXSnNhV05MWlhraU9pSXRMUzB0TFVKRlIwbE9JRkJWUWt4SlF5QkxSVmt0TFMwdExWeHVUVWxKUWtscVFVNUNaMnR4YUd0cFJ6bDNNRUpCVVVWR1FVRlBRMEZST0VGTlNVbENRMmRMUTBGUlJVRjZkRUpDWjBkR1IxRkpTbWRvYUM5cGFFRnphRnh1UjFZeFJtbHRVMHRQZDJ0TFpHdHVZVWxKUVVOamFGUXJXVXd4UzFjeVZUbFhUamsyVTBzNVdIWjNWblZvVWxsbFlrSjFjRk0xT1RaQ1pFNXplVmRFYWx4dVJpOUVWVEpWV21sbVIycElNM0I2ZEdKdFQzSlFLMnBWWlRsUE0ydFdNVmd5Tnl0YVowaDBha3RPT0dwVFZrSmxSemwyTkZvd1ZGTXplR1EwZDFWSlpWeHVlVzlhYWs1TVdrUjVZVGRMVW5wcFNsWndLMWM0TkUweVNIZEZaamxwSzJseFZuWm1ZVEI0YUhwbFJFTTRWRGw2UmxWNFRFeERZa1Y0YVVOdEsybzVWRnh1VDNaeWFqWmphelpRZG1Zd1FYcHhRazlyWmxKdlFYbEVPWEZPUVM4NFRUQnVUR04xVTFkUWIwcDRja1pHVnpZelYwWnJZazVoT1VSVkwxQnNSVTFTZDF4dVEydFJTWFozS3poSWIydzFUUzlZZGtaM1VVNVZiM2REVnk5elJXeE9ORFkwZDBwNFVuTklUVk4xVkVkU2RVVTBjbGgyUWxkQk9FUlhjSEI1UWtwMmQxeHVNbEZKUkVGUlFVSmNiaTB0TFMwdFJVNUVJRkJWUWt4SlF5QkxSVmt0TFMwdExWeHVJaXdpYTJWNVUybG5ibUYwZFhKbElqb2laWGxLZW1GWFpIVlpXRkl4WTIxVmFVOXBTbXRVVm14dFZVaFNXbVJWVWxwWk0wWnlWbFJHVkZwRVFsSlNha0phVld4YWVFMHlPREZWYTBwdFVUQkdVbFpxUW05VmJUbHlXbXQwZDFkRVRtdFZWMVUwWlZOMGVHRldaSFJsYms0eVVqSjRiRTFyVm01WFJ6VlBXbGhhVVU1Rk1ESmxiVGxXWTFST1NGcEZWbTlaTW1RelQxVXhSazB5YUVKVmFteHZUa2RuZW1WVWFETmxTR1F4VG1rNVZrNTZUalpXVlhoRVdsZG9VR1F6YkhaWlZYZDZXa1pHVDFZeVdsRlRSRUpVWkd4c1JHTlhjSEpVYTJSUldqRmtOVk16VG5aVk1WRjNUakZXUjFkdVdqRlhiWFJzVjFWU2FtTlZSVEJhTTI4d1dtdDBkMDR4V2tOaWVscDZZMnBLZUdGclNUTlNNR00xV1RKT1ZsVlhOVFJsUlRGVlUyMXdhRTlYVGs1VVYyUk1VMnhCZGxGcVl6UlZWM1J0VFVaYVExZFZVbmhOYTJ3MVYyNW9SbVJxUm5KV2FtTjJUREkxTmxOVmVFbE1NR1J4V1RCMFJrNUhNV3BYVlZrd1VteHNlVlZGZUZabGJrSmFVakZhYzJOVVVsaFpNRnBoVW0xb2NWcEhkRk5rTVVaRlRrZHJjbEV4YUhaa01GcDBWV3BDY1ZreWNGZFdNa2wzVmpCU1dHUlhkSFJVYkU1TlRXNVdjR0ZGVGpOalJXeHNVekJHV1U1dVdrTlhiVGxEVTI1YVdHRXlhRTlXUm1SMFYwVnNNMVJIVmtkVFYwcExaRlZWTUUweFJUbFFVMGx6U1cxa2MySXlTbWhpUlhSc1pWVnNhMGxxYjJsTlYxRjZXbXBrYlU1dFNURk5SR040VGtkYWJFNHlTVFJQVkZVeFRsUlNhMXBFV1RGT2VtTjZXV3BCYVdaUlBUMGlmUT09In0=`

	licenseFile, err := ioutil.TempFile("", "license")
	require.NoError(t, err)
	_, err = licenseFile.Write([]byte(data))
	require.NoError(t, err)
	err = licenseFile.Close()
	require.NoError(t, err)

	defer os.RemoveAll(licenseFile.Name())

	rootDir := t.TempDir()

	pullOptions := PullOptions{
		RootDir:   rootDir,
		Namespace: "default",
		Downstreams: []string{
			"this-cluster",
		},
		LicenseFile: licenseFile.Name(),
		AirgapRoot:  rootDir,
	}

	_, err = Pull("replicated://myapp", pullOptions)
	require.Error(t, err)

	// require.IsType(t, util.ActionableError{}, errors.Cause(err))
	// require.True(t, strings.Contains(err.Error(), "expired"), "error must contain expired")
}

func TestPullApplicationMetadata(t *testing.T) {
	tests := []struct {
		name               string
		upstreamURI        string
		license            *licensewrapper.LicenseWrapper
		versionLabel       string
		mockServerHandler  http.HandlerFunc
		expectedError      bool
		expectedErrorMsg   string
		expectNilMetadata  bool
		validateResult     func(t *testing.T, metadata interface{})
	}{
		{
			name:        "Valid replicated URI with v1beta1 license returns metadata",
			upstreamURI: "replicated://test-app",
			license: &licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID:       "test-license-id",
						AppSlug:         "test-app",
						LicenseSequence: 1,
						Endpoint:        "", // Will be set by test server
					},
				},
			},
			versionLabel: "1.0.0",
			mockServerHandler: func(w http.ResponseWriter, r *http.Request) {
				// Return metadata endpoint response
				if r.URL.Path == "/metadata/test-app" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: test-application
spec:
  title: "Test Application"
  icon: https://example.com/icon.png`))
					return
				}
				// Return empty branding
				if r.URL.Path == "/branding/test-app" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			},
			expectedError:     false,
			expectNilMetadata: false,
			validateResult: func(t *testing.T, metadata interface{}) {
				require.NotNil(t, metadata)
			},
		},
		{
			name:        "Valid replicated URI with v1beta2 license returns metadata",
			upstreamURI: "replicated://test-app-v2",
			license: &licensewrapper.LicenseWrapper{
				V2: &kotsv1beta2.License{
					Spec: kotsv1beta2.LicenseSpec{
						LicenseID:       "test-license-id-v2",
						AppSlug:         "test-app-v2",
						LicenseSequence: 1,
						Endpoint:        "", // Will be set by test server
					},
				},
			},
			versionLabel: "2.0.0",
			mockServerHandler: func(w http.ResponseWriter, r *http.Request) {
				// Return metadata endpoint response
				if r.URL.Path == "/metadata/test-app-v2" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: test-application-v2
spec:
  title: "Test Application V2"
  icon: https://example.com/icon.png`))
					return
				}
				// Return empty branding
				if r.URL.Path == "/branding/test-app-v2" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			},
			expectedError:     false,
			expectNilMetadata: false,
			validateResult: func(t *testing.T, metadata interface{}) {
				require.NotNil(t, metadata)
			},
		},
		{
			name:              "Invalid URI format returns error",
			upstreamURI:       "not a valid uri with spaces",
			license:           &licensewrapper.LicenseWrapper{V1: &kotsv1beta1.License{}},
			versionLabel:      "",
			mockServerHandler: nil, // No server needed for this test
			expectedError:     true,
			expectedErrorMsg:  "failed to parse uri",
			expectNilMetadata: true,
		},
		{
			name:        "Non-replicated URI scheme returns nil metadata without error",
			upstreamURI: "helm://stable/mysql",
			license: &licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						Endpoint: "https://replicated.app",
					},
				},
			},
			versionLabel:      "",
			mockServerHandler: nil, // No server needed for this test
			expectedError:     false,
			expectNilMetadata: true,
		},
		{
			name:        "Git scheme returns nil metadata without error",
			upstreamURI: "git://github.com/user/repo",
			license: &licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						Endpoint: "https://replicated.app",
					},
				},
			},
			versionLabel:      "",
			mockServerHandler: nil,
			expectedError:     false,
			expectNilMetadata: true,
		},
		{
			name:        "HTTP scheme returns nil metadata without error",
			upstreamURI: "http://example.com/app",
			license: &licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						Endpoint: "https://replicated.app",
					},
				},
			},
			versionLabel:      "",
			mockServerHandler: nil,
			expectedError:     false,
			expectNilMetadata: true,
		},
		{
			name:        "Error from GetApplicationMetadata is properly wrapped",
			upstreamURI: "replicated://error-app",
			license: &licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID:       "test-license-id",
						AppSlug:         "error-app",
						LicenseSequence: 1,
						Endpoint:        "", // Will be set by test server
					},
				},
			},
			versionLabel: "",
			mockServerHandler: func(w http.ResponseWriter, r *http.Request) {
				// Return 500 error
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectedError:     true,
			expectedErrorMsg:  "failed to get application metadata",
			expectNilMetadata: true,
		},
		{
			name:        "Replicated URI with channel returns metadata",
			upstreamURI: "replicated://test-app/beta",
			license: &licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID:       "test-license-id",
						AppSlug:         "test-app",
						LicenseSequence: 1,
						Endpoint:        "",
					},
				},
			},
			versionLabel: "1.0.0-beta.1",
			mockServerHandler: func(w http.ResponseWriter, r *http.Request) {
				// Handle metadata for beta channel
				if r.URL.Path == "/metadata/test-app/beta" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: test-application
spec:
  title: "Test Application Beta"
  icon: https://example.com/icon.png`))
					return
				}
				// Return empty branding
				if r.URL.Path == "/branding/test-app/beta" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			},
			expectedError:     false,
			expectNilMetadata: false,
			validateResult: func(t *testing.T, metadata interface{}) {
				require.NotNil(t, metadata)
			},
		},
		{
			name:        "Empty version label is handled correctly",
			upstreamURI: "replicated://test-app",
			license: &licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						LicenseID:       "test-license-id",
						AppSlug:         "test-app",
						LicenseSequence: 1,
						Endpoint:        "",
					},
				},
			},
			versionLabel: "", // Empty version label
			mockServerHandler: func(w http.ResponseWriter, r *http.Request) {
				// Should work without version label
				if r.URL.Path == "/metadata/test-app" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: test-application
spec:
  title: "Test Application"
  icon: https://example.com/icon.png`))
					return
				}
				if r.URL.Path == "/branding/test-app" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			},
			expectedError:     false,
			expectNilMetadata: false,
			validateResult: func(t *testing.T, metadata interface{}) {
				require.NotNil(t, metadata)
			},
		},
		{
			name:        "Empty license wrapper is handled",
			upstreamURI: "replicated://test-app",
			license:     &licensewrapper.LicenseWrapper{}, // Empty wrapper
			versionLabel: "1.0.0",
			mockServerHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: test-application
spec:
  title: "Test Application"
  icon: https://example.com/icon.png`))
			},
			expectedError:     false,
			expectNilMetadata: false,
			validateResult: func(t *testing.T, metadata interface{}) {
				require.NotNil(t, metadata)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server if handler is provided
			var server *httptest.Server
			if tt.mockServerHandler != nil {
				server = httptest.NewServer(tt.mockServerHandler)
				defer server.Close()

				// Set the endpoint in the license to use the test server
				if tt.license != nil {
					if tt.license.V1 != nil {
						tt.license.V1.Spec.Endpoint = server.URL
					} else if tt.license.V2 != nil {
						tt.license.V2.Spec.Endpoint = server.URL
					}
				}

				// Override environment variable to use test server
				t.Setenv("REPLICATED_APP_ENDPOINT", server.URL)
			}

			// Call the function
			metadata, err := PullApplicationMetadata(tt.upstreamURI, tt.license, tt.versionLabel)

			// Validate error expectations
			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				require.NoError(t, err)
			}

			// Validate metadata expectations
			if tt.expectNilMetadata {
				assert.Nil(t, metadata)
			} else if !tt.expectedError {
				assert.NotNil(t, metadata)
			}

			// Run custom validation if provided
			if tt.validateResult != nil && !tt.expectedError {
				tt.validateResult(t, metadata)
			}
		})
	}
}

func TestPullApplicationMetadata_NilLicense(t *testing.T) {
	// Test with nil license - should still work if non-replicated scheme
	metadata, err := PullApplicationMetadata("helm://stable/mysql", nil, "")
	require.NoError(t, err)
	assert.Nil(t, metadata)
}

func TestPullApplicationMetadata_URLParsing(t *testing.T) {
	tests := []struct {
		name          string
		upstreamURI   string
		shouldError   bool
		expectedError string
	}{
		{
			name:        "Valid replicated URL",
			upstreamURI: "replicated://my-app",
			shouldError: false,
		},
		{
			name:        "Valid replicated URL with channel",
			upstreamURI: "replicated://my-app/stable",
			shouldError: false,
		},
		{
			name:          "Invalid URL with spaces",
			upstreamURI:   "replicated://my app with spaces",
			shouldError:   true,
			expectedError: "failed to parse uri",
		},
		{
			name:          "Invalid URL format",
			upstreamURI:   "://invalid",
			shouldError:   true,
			expectedError: "failed to parse uri",
		},
		{
			name:        "Valid HTTP URL",
			upstreamURI: "http://example.com/app",
			shouldError: false,
		},
		{
			name:        "Valid HTTPS URL",
			upstreamURI: "https://example.com/app",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			license := &licensewrapper.LicenseWrapper{
				V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						Endpoint: "https://replicated.app",
					},
				},
			}

			_, err := PullApplicationMetadata(tt.upstreamURI, license, "")

			if tt.shouldError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				// For valid non-replicated schemes, we expect no error but nil metadata
				require.NoError(t, err)
			}
		})
	}
}
