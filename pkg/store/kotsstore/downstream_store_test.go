package kotsstore

import (
	"encoding/json"
	"testing"

	"github.com/blang/semver"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/store/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_isSameUpstreamRelease(t *testing.T) {
	tests := []struct {
		name             string
		v1               *downstreamtypes.DownstreamVersion
		v2               *downstreamtypes.DownstreamVersion
		isSemverRequired bool
		expected         bool
	}{
		{
			name: "non-semver, same channel, different cursor",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "3",
			},
			expected: false,
		},
		{
			name: "non-semver, same cursor, different channel",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-2",
				UpdateCursor: "2",
			},
			expected: false,
		},
		{
			name: "non-semver, same cursor, same channel",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
			},
			expected: true,
		},
		{
			name: "semver, same channel, same cursor, same semver",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "3.0",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "3.0",
			},
			isSemverRequired: true,
			expected:         true,
		},
		{
			name: "semver, same channel, same cursor, different semver",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "3.0",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "4.0",
			},
			isSemverRequired: true,
			expected:         true,
		},
		{
			name: "semver, different channel, same cursor, different semver",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "3.0",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-2",
				UpdateCursor: "2",
				VersionLabel: "4.0",
			},
			isSemverRequired: true,
			expected:         false,
		},
		{
			name: "semver, different channel, same cursor, same semver",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "4.0",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-2",
				UpdateCursor: "2",
				VersionLabel: "4.0",
			},
			isSemverRequired: true,
			expected:         true,
		},
		{
			name: "semver, same channel, different cursor, different semver",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "3.0",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "3",
				VersionLabel: "4.0",
			},
			isSemverRequired: true,
			expected:         false,
		},
		{
			name: "semver, same channel, different cursor, same semver",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "3.0",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "3",
				VersionLabel: "3.0",
			},
			isSemverRequired: true,
			expected:         true,
		},
		{
			name: "semver, different channel, different cursor, same semver",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "4.0",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-2",
				UpdateCursor: "3",
				VersionLabel: "4.0",
			},
			isSemverRequired: true,
			expected:         true,
		},
		{
			name: "semver, different channel, different cursor, different semver",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "3.0",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-2",
				UpdateCursor: "3",
				VersionLabel: "4.0",
			},
			isSemverRequired: true,
			expected:         false,
		},
		{
			name: "semver and non-semver, same channel, same cursor",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "3.0",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
			},
			isSemverRequired: true,
			expected:         true,
		},
		{
			name: "semver and non-semver, different channel, same cursor",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "3.0",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-2",
				UpdateCursor: "2",
			},
			isSemverRequired: true,
			expected:         false,
		},
		{
			name: "semver and non-semver, same channel, different cursor",
			v1: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "3.0",
			},
			v2: &downstreamtypes.DownstreamVersion{
				ChannelID:    "channel-id-1",
				UpdateCursor: "3",
			},
			isSemverRequired: true,
			expected:         false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v1c := cursor.MustParse(test.v1.UpdateCursor)
			test.v1.Cursor = &v1c

			v2c := cursor.MustParse(test.v2.UpdateCursor)
			test.v2.Cursor = &v2c

			v1s, err := semver.ParseTolerant(test.v1.VersionLabel)
			if err == nil {
				test.v1.Semver = &v1s
			}

			v2s, err := semver.ParseTolerant(test.v2.VersionLabel)
			if err == nil {
				test.v2.Semver = &v2s
			}

			same := isSameUpstreamRelease(test.v1, test.v2, test.isSemverRequired)
			assert.Equal(t, test.expected, same)
		})
	}
}

func Test_isAppVersionDeployable(t *testing.T) {
	tests := []struct {
		name                 string
		version              *downstreamtypes.DownstreamVersion
		appVersions          *downstreamtypes.DownstreamVersions
		isSemverRequired     bool
		currentECConfig      *embeddedclusterv1beta1.Config
		versionECConfig      *embeddedclusterv1beta1.Config
		setup                func(t *testing.T)
		expectedIsDeployable bool
		expectedCause        string
	}{
		{
			name: "failing strict preflights",
			version: &downstreamtypes.DownstreamVersion{
				HasFailingStrictPreflights: true,
			},
			appVersions:          &downstreamtypes.DownstreamVersions{},
			expectedIsDeployable: false,
			expectedCause:        "Deployment is disabled as a strict analyzer in this version's preflight checks has failed or has not been run.",
		},
		{
			name: "pending download",
			version: &downstreamtypes.DownstreamVersion{
				Status: types.VersionPendingDownload,
			},
			appVersions:          &downstreamtypes.DownstreamVersions{},
			expectedIsDeployable: false,
			expectedCause:        "Version is pending download.",
		},
		{
			name: "pending config",
			version: &downstreamtypes.DownstreamVersion{
				Status: types.VersionPendingConfig,
			},
			appVersions:          &downstreamtypes.DownstreamVersions{},
			expectedIsDeployable: false,
			expectedCause:        "Version is pending configuration.",
		},
		{
			name:                 "no version is deployed yet",
			version:              &downstreamtypes.DownstreamVersion{},
			appVersions:          &downstreamtypes.DownstreamVersions{},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "version is same as deployed version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence: 7,
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence: 7,
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "rollback is determined by latest downloaded version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence: 0,
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence: 3,
						Status:   types.VersionPendingDownload,
					},
					{
						Sequence: 2,
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence: 1,
					},
					{
						Sequence: 0,
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence: 1,
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		/* ---- Non semver tests begin here ---- */
		{
			name: "non-semver -- deployed version is from a different channel, not required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:  1,
				ChannelID: "channel-id-1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:  1,
						ChannelID: "channel-id-1",
					},
					{
						Sequence:  0,
						ChannelID: "channel-id-2",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:  0,
					ChannelID: "channel-id-2",
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- deployed version is from a different channel, is required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:  1,
				ChannelID: "channel-id-1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:  1,
						ChannelID: "channel-id-1",
					},
					{
						Sequence:   0,
						ChannelID:  "channel-id-2",
						IsRequired: true,
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:   0,
					ChannelID:  "channel-id-2",
					IsRequired: true,
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- deployed version is from a different channel, is required, required releases in between from different channel",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:  3,
				ChannelID: "channel-id-1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:  3,
						ChannelID: "channel-id-1",
					},
					{
						Sequence:   2,
						ChannelID:  "channel-id-2",
						IsRequired: true,
					},
					{
						Sequence:   1,
						ChannelID:  "channel-id-2",
						IsRequired: true,
					},
					{
						Sequence:   0,
						ChannelID:  "channel-id-2",
						IsRequired: true,
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:   0,
					ChannelID:  "channel-id-2",
					IsRequired: true,
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- deployed version is from a different channel, not required, required releases in between from same channel, same variants as version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     3,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "3",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     0,
					ChannelID:    "channel-id-2",
					UpdateCursor: "1",
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- deployed version is from a different channel, not required, required releases in between from same channel, different variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     3,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "4",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     0,
					ChannelID:    "channel-id-2",
					UpdateCursor: "1",
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- deployed version is from same channel, not required, required releases in between from same channel, same variants as version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     3,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "2",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     0,
					ChannelID:    "channel-id-1",
					UpdateCursor: "1",
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- deployed version is from same channel, not required, required releases in between from same channel, same variants as deployed version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     3,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "2",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "1",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "1",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     0,
					ChannelID:    "channel-id-1",
					IsRequired:   true,
					UpdateCursor: "1",
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- deployed version is from same channel, is required, required releases in between from same channel, different variants, lower cursor",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     4,
				ChannelID:    "channel-id-1",
				UpdateCursor: "5",
				VersionLabel: "5.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						UpdateCursor: "5",
						VersionLabel: "5.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						IsRequired:   true,
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
					IsRequired:   true,
					VersionLabel: "2.0",
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "This version cannot be deployed because versions 3.0, 4.0 are required and must be deployed first.",
		},
		{
			name: "non-semver -- deployed version is from same channel, is required, required releases in between from same channel, different variants, 2 higher cursor and 1 lower cursor from different channel",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     5,
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				IsRequired:   true,
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     5,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						IsRequired:   true,
						VersionLabel: "2.0",
					},
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						VersionLabel: "4.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "3.1",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						IsRequired:   true,
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
					IsRequired:   true,
					VersionLabel: "2.0",
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- deployed version is from same channel, not required, required and non-required releases in between from same channel, different variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     5,
				ChannelID:    "channel-id-1",
				UpdateCursor: "6",
				VersionLabel: "6.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     5,
						ChannelID:    "channel-id-1",
						UpdateCursor: "6",
						VersionLabel: "6.0",
					},
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "5",
						VersionLabel: "5.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
					VersionLabel: "2.0",
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "This version cannot be deployed because versions 3.0, 5.0 are required and must be deployed first.",
		},
		{
			name: "non-semver -- deployed version is from same channel, not required, required releases in between from same channel and same variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     5,
				ChannelID:    "channel-id-1",
				UpdateCursor: "6",
				VersionLabel: "4.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     5,
						ChannelID:    "channel-id-1",
						UpdateCursor: "6",
						VersionLabel: "4.0",
					},
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
					VersionLabel: "2.0",
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "This version cannot be deployed because version 3.0 is required and must be deployed first.",
		},
		/* ---- Non semver rollback tests begin here ---- */
		{
			name: "non-semver -- disabled rollback -- deployed version is from a different channel, not required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "Rollback is not supported.",
		},
		{
			name: "non-semver -- disabled rollback for latest version, enabled rollback for deployed version -- deployed version is from a different channel, not required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "1",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "Rollback is not supported.",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from a different channel, not required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- disable rollback -- deployed version is from a different channel, is required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "2",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-2",
					IsRequired:   true,
					UpdateCursor: "2",
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "Rollback is not supported.",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from a different channel, is required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "2",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-2",
					IsRequired:   true,
					UpdateCursor: "2",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from a different channel, is required, required releases in between from same channel as version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "1",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "2",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     2,
					ChannelID:    "channel-id-1",
					IsRequired:   true,
					UpdateCursor: "1",
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from a different channel, not required, required releases in between from same channel, same variants as version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "3",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     3,
					ChannelID:    "channel-id-2",
					UpdateCursor: "1",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from a different channel, not required, required releases in between from same channel, different variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "2",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     3,
					ChannelID:    "channel-id-2",
					UpdateCursor: "1",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, is required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					IsRequired:   true,
					UpdateCursor: "2",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, is required, required releases in between from different channel",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "2",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     2,
					ChannelID:    "channel-id-2",
					IsRequired:   true,
					UpdateCursor: "2",
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from different channel",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-2",
						UpdateCursor: "2",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     2,
					ChannelID:    "channel-id-2",
					UpdateCursor: "2",
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel, same variants as version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "3",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     3,
					ChannelID:    "channel-id-1",
					UpdateCursor: "4",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, is required, required releases in between from same channel, same variants as deployed version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "3",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "3",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     3,
					ChannelID:    "channel-id-1",
					IsRequired:   true,
					UpdateCursor: "4",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel, different variants, lower cursor, same version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "4",
				VersionLabel: "4.0",
				KOTSKinds: &kotsutil.KotsKinds{
					KotsApplication: kotsv1beta1.Application{
						Spec: kotsv1beta1.ApplicationSpec{
							AllowRollback: true,
						},
					},
				},
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "3",
						IsRequired:   true,
						VersionLabel: "3.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     4,
					ChannelID:    "channel-id-1",
					UpdateCursor: "4",
					VersionLabel: "4.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel, different variants, lower cursor, different version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "1",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						UpdateCursor: "5",
						VersionLabel: "5.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						IsRequired:   true,
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     4,
					ChannelID:    "channel-id-1",
					UpdateCursor: "5",
					VersionLabel: "5.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel, different variants, 2 higher cursor and 1 lower cursor from different channel, same version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "4",
				VersionLabel: "4.0",
				KOTSKinds: &kotsutil.KotsKinds{
					KotsApplication: kotsv1beta1.Application{
						Spec: kotsv1beta1.ApplicationSpec{
							AllowRollback: true,
						},
					},
				},
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "6",
						VersionLabel: "6.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "5",
						IsRequired:   true,
						VersionLabel: "5.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     4,
					ChannelID:    "channel-id-1",
					UpdateCursor: "4",
					VersionLabel: "4.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel, different variants, 2 higher cursor and 1 lower cursor from different channel, different version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						UpdateCursor: "5",
						VersionLabel: "5.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						IsRequired:   true,
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     4,
					ChannelID:    "channel-id-1",
					UpdateCursor: "5",
					VersionLabel: "5.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel, different variants, 2 higher cursor than deployed and 1 lower cursor from different channel, different version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						UpdateCursor: "5",
						VersionLabel: "5.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "6",
						IsRequired:   true,
						VersionLabel: "6.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "7",
						VersionLabel: "7.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     4,
					ChannelID:    "channel-id-1",
					UpdateCursor: "5",
					VersionLabel: "5.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, not required, required and non-required releases in between from same channel, different variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     1,
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     5,
						ChannelID:    "channel-id-1",
						UpdateCursor: "6",
						VersionLabel: "6.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "5",
						VersionLabel: "5.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     5,
					ChannelID:    "channel-id-1",
					UpdateCursor: "6",
					VersionLabel: "6.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "non-semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel and same variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     1,
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     5,
						ChannelID:    "channel-id-1",
						UpdateCursor: "6",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     5,
					ChannelID:    "channel-id-1",
					UpdateCursor: "6",
					VersionLabel: "4.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		/* ---- Non semver rollback tests end here ---- */
		/* ---- Non semver tests end here ---- */

		/* ---- Semver tests begin here ---- */
		{
			name: "semver -- deployed version is from a different channel, not required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     1,
				ChannelID:    "channel-id-1",
				UpdateCursor: "1",
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     0,
					ChannelID:    "channel-id-2",
					UpdateCursor: "1",
					VersionLabel: "1.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "semver -- deployed version is from a different channel, is required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     1,
				ChannelID:    "channel-id-1",
				UpdateCursor: "1",
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     0,
					ChannelID:    "channel-id-2",
					IsRequired:   true,
					UpdateCursor: "1",
					VersionLabel: "1.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "semver -- deployed version is from a different channel, is required, required releases in between from different channel",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     3,
				ChannelID:    "channel-id-1",
				UpdateCursor: "4",
				VersionLabel: "4.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     0,
					ChannelID:    "channel-id-2",
					IsRequired:   true,
					UpdateCursor: "1",
					VersionLabel: "1.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "This version cannot be deployed because versions 2.0, 3.0 are required and must be deployed first.",
		},
		{
			name: "semver -- deployed version is from a different channel, not required, required releases in between from same channel, same variants as version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     3,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "2",
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     0,
					ChannelID:    "channel-id-2",
					UpdateCursor: "1",
					VersionLabel: "1.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "semver -- deployed version is from a different channel, not required, required releases in between from same channel, different variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     3,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "4",
				VersionLabel: "4.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     0,
					ChannelID:    "channel-id-2",
					UpdateCursor: "1",
					VersionLabel: "1.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "This version cannot be deployed because versions 2.0, 3.0 are required and must be deployed first.",
		},
		{
			name: "semver -- deployed version is from same channel, not required, required releases in between from same channel, same variants as version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     3,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "2",
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     0,
					ChannelID:    "channel-id-1",
					UpdateCursor: "1",
					VersionLabel: "1.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "semver -- deployed version is from same channel, not required, required releases in between from same channel, same variants as deployed version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     3,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "2",
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     0,
					ChannelID:    "channel-id-1",
					IsRequired:   true,
					UpdateCursor: "1",
					VersionLabel: "1.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "semver -- deployed version is from same channel, is required, required releases in between from same channel, different variants, lower cursor",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     4,
				ChannelID:    "channel-id-1",
				UpdateCursor: "5",
				VersionLabel: "5.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						UpdateCursor: "5",
						VersionLabel: "5.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						IsRequired:   true,
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
					IsRequired:   true,
					VersionLabel: "2.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "This version cannot be deployed because versions 3.0, 4.0 are required and must be deployed first.",
		},
		{
			name: "semver -- deployed version is from same channel, is required, required releases in between from same channel, different variants, 2 higher cursor and 1 lower cursor from different channel",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     5,
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				IsRequired:   true,
				VersionLabel: "5.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     5,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						IsRequired:   true,
						VersionLabel: "5.0",
					},
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						VersionLabel: "4.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "3.1",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						IsRequired:   true,
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
					IsRequired:   true,
					VersionLabel: "2.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "This version cannot be deployed because versions 3.0, 3.1, 4.0 are required and must be deployed first.",
		},
		{
			name: "semver -- deployed version is from same channel, not required, required and non-required releases in between from same channel, different variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     5,
				ChannelID:    "channel-id-1",
				UpdateCursor: "6",
				VersionLabel: "6.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     5,
						ChannelID:    "channel-id-1",
						UpdateCursor: "6",
						VersionLabel: "6.0",
					},
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "5",
						VersionLabel: "5.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
					VersionLabel: "2.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "This version cannot be deployed because versions 3.0, 5.0 are required and must be deployed first.",
		},
		{
			name: "semver -- deployed version is from same channel, not required, required releases in between from same channel and same variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     5,
				ChannelID:    "channel-id-1",
				UpdateCursor: "6",
				VersionLabel: "4.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     5,
						ChannelID:    "channel-id-1",
						UpdateCursor: "6",
						VersionLabel: "4.0",
					},
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
					VersionLabel: "2.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "This version cannot be deployed because version 3.0 is required and must be deployed first.",
		},
		/* ---- Semver rollback tests begin here ---- */
		{
			name: "semver -- disabled rollback -- deployed version is from a different channel, not required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
					VersionLabel: "2.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "Rollback is not supported.",
		},
		{
			name: "semver -- disabled rollback for latest version, enabled rollback for deployed version -- deployed version is from a different channel, not required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						VersionLabel: "2.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "1",
					VersionLabel: "2.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "Rollback is not supported.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from a different channel, not required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					UpdateCursor: "2",
					VersionLabel: "2.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "semver -- disable rollback -- deployed version is from a different channel, is required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "1",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-2",
					IsRequired:   true,
					UpdateCursor: "2",
					VersionLabel: "2.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "Rollback is not supported.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from a different channel, is required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "1",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-2",
					IsRequired:   true,
					UpdateCursor: "2",
					VersionLabel: "2.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from a different channel, is required, required releases in between from same channel as version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     2,
					ChannelID:    "channel-id-1",
					IsRequired:   true,
					UpdateCursor: "1",
					VersionLabel: "3.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from a different channel, not required, required releases in between from same channel, same variants as version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "3",
				VersionLabel: "3.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     3,
					ChannelID:    "channel-id-2",
					UpdateCursor: "1",
					VersionLabel: "4.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "semver -- allow rollback -- deployed version is from a different channel, not required, required releases in between from same channel, different variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "2",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     3,
					ChannelID:    "channel-id-2",
					UpdateCursor: "1",
					VersionLabel: "4.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from same channel, is required, no required releases in between",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "1",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     1,
					ChannelID:    "channel-id-1",
					IsRequired:   true,
					UpdateCursor: "2",
					VersionLabel: "2.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from same channel, is required, required releases in between from different channel",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     2,
					ChannelID:    "channel-id-2",
					IsRequired:   true,
					UpdateCursor: "2",
					VersionLabel: "3.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from different channel",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-2",
				UpdateCursor: "1",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-2",
						UpdateCursor: "2",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-2",
						UpdateCursor: "1",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     2,
					ChannelID:    "channel-id-2",
					UpdateCursor: "2",
					VersionLabel: "3.0",
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel, same variants as version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				IsRequired:   true,
				UpdateCursor: "3",
				VersionLabel: "3.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     3,
					ChannelID:    "channel-id-1",
					UpdateCursor: "4",
					VersionLabel: "4.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "semver -- allow rollback -- deployed version is from same channel, is required, required releases in between from same channel, same variants as deployed version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "3",
				VersionLabel: "3.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "4",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     3,
					ChannelID:    "channel-id-1",
					IsRequired:   true,
					UpdateCursor: "4",
					VersionLabel: "4.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel, different variants, lower cursor",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "5",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						UpdateCursor: "3",
						IsRequired:   true,
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "5",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     3,
					ChannelID:    "channel-id-1",
					UpdateCursor: "4",
					VersionLabel: "4.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel, different variants, 2 higher cursor and 1 lower cursor from different channel",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						UpdateCursor: "5",
						VersionLabel: "5.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						IsRequired:   true,
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     4,
					ChannelID:    "channel-id-1",
					UpdateCursor: "5",
					VersionLabel: "5.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel, different variants, 2 higher cursor than deployed and 1 lower cursor from different channel, different version",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     0,
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "1.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						UpdateCursor: "5",
						VersionLabel: "5.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "6",
						IsRequired:   true,
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "7",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-2",
						IsRequired:   true,
						UpdateCursor: "1",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     4,
					ChannelID:    "channel-id-1",
					UpdateCursor: "5",
					VersionLabel: "5.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from same channel, not required, required and non-required releases in between from same channel, different variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     1,
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     5,
						ChannelID:    "channel-id-1",
						UpdateCursor: "6",
						VersionLabel: "6.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "5",
						VersionLabel: "5.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						UpdateCursor: "4",
						VersionLabel: "4.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     5,
					ChannelID:    "channel-id-1",
					UpdateCursor: "6",
					VersionLabel: "6.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		{
			name: "semver -- allow rollback -- deployed version is from same channel, not required, required releases in between from same channel and same variants",
			version: &downstreamtypes.DownstreamVersion{
				Sequence:     1,
				ChannelID:    "channel-id-1",
				UpdateCursor: "2",
				VersionLabel: "2.0",
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						Sequence:     5,
						ChannelID:    "channel-id-1",
						UpdateCursor: "6",
						VersionLabel: "4.0",
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						Sequence:     4,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     3,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     2,
						ChannelID:    "channel-id-1",
						IsRequired:   true,
						UpdateCursor: "3",
						VersionLabel: "3.0",
					},
					{
						Sequence:     1,
						ChannelID:    "channel-id-1",
						UpdateCursor: "2",
						VersionLabel: "2.0",
					},
					{
						Sequence:     0,
						ChannelID:    "channel-id-1",
						UpdateCursor: "1",
						IsRequired:   true,
						VersionLabel: "1.0",
					},
				},
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					Sequence:     5,
					ChannelID:    "channel-id-1",
					UpdateCursor: "6",
					VersionLabel: "4.0",
					KOTSKinds: &kotsutil.KotsKinds{
						KotsApplication: kotsv1beta1.Application{
							Spec: kotsv1beta1.ApplicationSpec{
								AllowRollback: true,
							},
						},
					},
				},
			},
			isSemverRequired:     true,
			expectedIsDeployable: false,
			expectedCause:        "One or more non-reversible versions have been deployed since this version.",
		},
		/* ---- Semver rollback tests end here ---- */
		/* ---- Semver tests end here ---- */
		/* ---- Embedded cluster config tests start here ---- */
		{
			name: "embedded cluster config change should not allow rollbacks",
			setup: func(t *testing.T) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "1234")
			},
			version: &downstreamtypes.DownstreamVersion{
				VersionLabel: "1.0.0",
				Sequence:     0,
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					VersionLabel: "2.0.0",
					Sequence:     1,
				},
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						VersionLabel: "3.0.0",
						Sequence:     2,
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						VersionLabel: "2.0.0",
						Sequence:     1,
					},
					{
						VersionLabel: "1.0.0",
						Sequence:     0,
					},
				},
			},
			currentECConfig: &embeddedclusterv1beta1.Config{
				Spec: embeddedclusterv1beta1.ConfigSpec{
					Version: "1.0.0-ec.1",
				},
			},
			versionECConfig: &embeddedclusterv1beta1.Config{
				Spec: embeddedclusterv1beta1.ConfigSpec{
					Version: "1.0.0-ec.0",
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "Rollback is not supported, cluster configuration has changed.",
		},
		{
			name: "embedded cluster config no change should allow rollbacks",
			setup: func(t *testing.T) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "1234")
			},
			version: &downstreamtypes.DownstreamVersion{
				VersionLabel: "1.0.0",
				Sequence:     0,
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					VersionLabel: "2.0.0",
					Sequence:     1,
				},
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						VersionLabel: "3.0.0",
						Sequence:     2,
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: true,
								},
							},
						},
					},
					{
						VersionLabel: "2.0.0",
						Sequence:     1,
					},
					{
						VersionLabel: "1.0.0",
						Sequence:     0,
					},
				},
			},
			currentECConfig: &embeddedclusterv1beta1.Config{
				Spec: embeddedclusterv1beta1.ConfigSpec{
					Version: "1.0.0-ec.1",
				},
			},
			versionECConfig: &embeddedclusterv1beta1.Config{
				Spec: embeddedclusterv1beta1.ConfigSpec{
					Version: "1.0.0-ec.1",
				},
			},
			expectedIsDeployable: true,
			expectedCause:        "",
		},
		{
			name: "embedded cluster, allowRollback = false should not allow rollbacks",
			setup: func(t *testing.T) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "1234")
			},
			version: &downstreamtypes.DownstreamVersion{
				VersionLabel: "1.0.0",
				Sequence:     0,
			},
			appVersions: &downstreamtypes.DownstreamVersions{
				CurrentVersion: &downstreamtypes.DownstreamVersion{
					VersionLabel: "2.0.0",
					Sequence:     1,
				},
				AllVersions: []*downstreamtypes.DownstreamVersion{
					{
						VersionLabel: "3.0.0",
						Sequence:     2,
						KOTSKinds: &kotsutil.KotsKinds{
							KotsApplication: kotsv1beta1.Application{
								Spec: kotsv1beta1.ApplicationSpec{
									AllowRollback: false,
								},
							},
						},
					},
					{
						VersionLabel: "2.0.0",
						Sequence:     1,
					},
					{
						VersionLabel: "1.0.0",
						Sequence:     0,
					},
				},
			},
			currentECConfig: &embeddedclusterv1beta1.Config{
				Spec: embeddedclusterv1beta1.ConfigSpec{
					Version: "1.0.0-ec.1",
				},
			},
			versionECConfig: &embeddedclusterv1beta1.Config{
				Spec: embeddedclusterv1beta1.ConfigSpec{
					Version: "1.0.0-ec.0",
				},
			},
			expectedIsDeployable: false,
			expectedCause:        "Rollback is not supported.",
		},
		/* ---- Embedded cluster config tests end here ---- */
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.version.UpdateCursor != "" {
				vc := cursor.MustParse(test.version.UpdateCursor)
				test.version.Cursor = &vc
			}
			vs, err := semver.ParseTolerant(test.version.VersionLabel)
			if err == nil {
				test.version.Semver = &vs
			}

			for _, v := range test.appVersions.AllVersions {
				if v.UpdateCursor != "" {
					c := cursor.MustParse(v.UpdateCursor)
					v.Cursor = &c
				}
				s, err := semver.ParseTolerant(v.VersionLabel)
				if err == nil {
					v.Semver = &s
				}
			}

			if test.appVersions.CurrentVersion != nil {
				if test.appVersions.CurrentVersion.UpdateCursor != "" {
					cvc := cursor.MustParse(test.appVersions.CurrentVersion.UpdateCursor)
					test.appVersions.CurrentVersion.Cursor = &cvc
				}
				cvs, err := semver.ParseTolerant(test.appVersions.CurrentVersion.VersionLabel)
				if err == nil {
					test.appVersions.CurrentVersion.Semver = &cvs
				}
			}

			var currentECConfig, versionECConfig []byte
			if test.currentECConfig != nil {
				currentECConfig, err = json.Marshal(test.currentECConfig)
				require.NoError(t, err)
			}
			if test.versionECConfig != nil {
				versionECConfig, err = json.Marshal(test.versionECConfig)
				require.NoError(t, err)
			}

			if test.setup != nil {
				test.setup(t)
			}

			isDeployable, cause := isAppVersionDeployable("APPID", test.version, test.appVersions, test.isSemverRequired, currentECConfig, versionECConfig)
			assert.Equal(t, test.expectedIsDeployable, isDeployable)
			assert.Equal(t, test.expectedCause, cause)
		})
	}
}
