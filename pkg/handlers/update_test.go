package handlers

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

func Test_getKotsUpgradeVersion(t *testing.T) {
	tests := []struct {
		name          string
		kotsKinds     *kotsutil.KotsKinds
		latestVersion string
		isError       bool
		want          string
	}{
		{
			name:          "feature not enabled",
			kotsKinds:     &kotsutil.KotsKinds{},
			latestVersion: "v3.0.0",
			isError:       true,
			want:          "",
		},
		{
			name: "no target version",
			kotsKinds: &kotsutil.KotsKinds{
				KotsApplication: kotsv1beta1.Application{
					Spec: kotsv1beta1.ApplicationSpec{
						ConsoleFeatureFlags: []string{"admin-console-auto-updates"},
					},
				},
			},
			latestVersion: "v3.0.0",
			isError:       true,
			want:          "",
		},
		{
			name: "min version only",
			kotsKinds: &kotsutil.KotsKinds{
				KotsApplication: kotsv1beta1.Application{
					Spec: kotsv1beta1.ApplicationSpec{
						ConsoleFeatureFlags: []string{"admin-console-auto-updates"},
						MinKotsVersion:      "v1.0.0",
					},
				},
			},
			latestVersion: "",
			isError:       false,
			want:          "v1.0.0",
		},
		{
			name: "min version only with latest",
			kotsKinds: &kotsutil.KotsKinds{
				KotsApplication: kotsv1beta1.Application{
					Spec: kotsv1beta1.ApplicationSpec{
						ConsoleFeatureFlags: []string{"admin-console-auto-updates"},
						MinKotsVersion:      "v1.0.0",
					},
				},
			},
			latestVersion: "v3.0.0",
			isError:       false,
			want:          "v3.0.0",
		},
		{
			name: "target version only",
			kotsKinds: &kotsutil.KotsKinds{
				KotsApplication: kotsv1beta1.Application{
					Spec: kotsv1beta1.ApplicationSpec{
						ConsoleFeatureFlags: []string{"admin-console-auto-updates"},
						TargetKotsVersion:   "v2.0.0",
					},
				},
			},
			latestVersion: "v3.0.0",
			isError:       false,
			want:          "v2.0.0",
		},
		{
			name: "min and target version with latest",
			kotsKinds: &kotsutil.KotsKinds{
				KotsApplication: kotsv1beta1.Application{
					Spec: kotsv1beta1.ApplicationSpec{
						ConsoleFeatureFlags: []string{"admin-console-auto-updates"},
						MinKotsVersion:      "v1.0.0",
						TargetKotsVersion:   "v2.0.0",
					},
				},
			},
			latestVersion: "v3.0.0",
			isError:       false,
			want:          "v2.0.0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := getKotsUpgradeVersion(test.kotsKinds, test.latestVersion)
			if test.isError {
				require.Error(t, err)
			} else {
				require.Equal(t, test.want, got)
			}
		})
	}

}
