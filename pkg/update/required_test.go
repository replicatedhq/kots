package update

import (
	"testing"

	"github.com/blang/semver"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/cursor"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

func Test_getRequiredAirgapUpdates(t *testing.T) {
	channelID := "channel-id"
	tests := []struct {
		name              string
		airgap            *kotsv1beta1.Airgap
		license           *kotsv1beta1.License
		installedVersions []*downstreamtypes.DownstreamVersion
		channelChanged    bool
		wantSemver        []string
		wantNoSemver      []string
	}{
		{
			name: "nothing is installed yet",
			airgap: &kotsv1beta1.Airgap{
				Spec: kotsv1beta1.AirgapSpec{
					ChannelID: channelID,
					RequiredReleases: []kotsv1beta1.AirgapReleaseMeta{
						{
							VersionLabel: "0.1.123",
							UpdateCursor: "123",
						},
					},
				},
			},
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{},
			},
			installedVersions: []*downstreamtypes.DownstreamVersion{},
			wantNoSemver:      []string{},
			wantSemver:        []string{},
		},
		{
			name: "latest satisfies all prerequsites",
			airgap: &kotsv1beta1.Airgap{
				Spec: kotsv1beta1.AirgapSpec{
					ChannelID: channelID,
					RequiredReleases: []kotsv1beta1.AirgapReleaseMeta{
						{
							VersionLabel: "0.1.123",
							UpdateCursor: "123",
						},
						{
							VersionLabel: "0.1.120",
							UpdateCursor: "120",
						},
						{
							VersionLabel: "0.1.115",
							UpdateCursor: "115",
						},
					},
				},
			},
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{},
			},
			installedVersions: []*downstreamtypes.DownstreamVersion{
				{
					ChannelID:    channelID,
					VersionLabel: "0.1.124",
					UpdateCursor: "124",
				},
			},
			wantNoSemver: []string{},
			wantSemver:   []string{},
		},
		{
			name: "need some prerequsites",
			airgap: &kotsv1beta1.Airgap{
				Spec: kotsv1beta1.AirgapSpec{
					ChannelID: channelID,
					RequiredReleases: []kotsv1beta1.AirgapReleaseMeta{
						{
							VersionLabel: "0.1.123",
							UpdateCursor: "123",
						},
						{
							VersionLabel: "0.1.120",
							UpdateCursor: "120",
						},
						{
							VersionLabel: "0.1.115",
							UpdateCursor: "115",
						},
					},
				},
			},
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{},
			},
			installedVersions: []*downstreamtypes.DownstreamVersion{
				{
					ChannelID:    channelID,
					VersionLabel: "0.1.117",
					UpdateCursor: "117",
				},
			},
			wantNoSemver: []string{"0.1.120", "0.1.123"},
			wantSemver:   []string{"0.1.120", "0.1.123"},
		},
		{
			name: "need all prerequsites",
			airgap: &kotsv1beta1.Airgap{
				Spec: kotsv1beta1.AirgapSpec{
					ChannelID: channelID,
					RequiredReleases: []kotsv1beta1.AirgapReleaseMeta{
						{
							VersionLabel: "0.1.123",
							UpdateCursor: "123",
						},
						{
							VersionLabel: "0.1.120",
							UpdateCursor: "120",
						},
						{
							VersionLabel: "0.1.115",
							UpdateCursor: "115",
						},
					},
				},
			},
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{},
			},
			installedVersions: []*downstreamtypes.DownstreamVersion{
				{
					ChannelID:    channelID,
					VersionLabel: "0.1.113",
					UpdateCursor: "113",
				},
			},
			wantNoSemver: []string{"0.1.115", "0.1.120", "0.1.123"},
			wantSemver:   []string{"0.1.115", "0.1.120", "0.1.123"},
		},
		{
			name: "check across multiple channels",
			airgap: &kotsv1beta1.Airgap{
				Spec: kotsv1beta1.AirgapSpec{
					ChannelID: channelID,
					RequiredReleases: []kotsv1beta1.AirgapReleaseMeta{
						{
							VersionLabel: "0.1.123",
							UpdateCursor: "123",
						},
						{
							VersionLabel: "0.1.120",
							UpdateCursor: "120",
						},
						{
							VersionLabel: "0.1.115",
							UpdateCursor: "115",
						},
					},
				},
			},
			license: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{},
			},
			channelChanged: true,
			installedVersions: []*downstreamtypes.DownstreamVersion{
				{
					ChannelID:    "different-channel",
					VersionLabel: "0.1.117",
					UpdateCursor: "117",
				},
			},
			wantNoSemver: []string{},
			wantSemver:   []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			for _, v := range tt.installedVersions {
				s := semver.MustParse(v.VersionLabel)
				v.Semver = &s

				c := cursor.MustParse(v.UpdateCursor)
				v.Cursor = &c
			}

			// cursor based
			tt.license.Spec.IsSemverRequired = false
			got, err := getRequiredAirgapUpdates(tt.airgap, tt.license, tt.installedVersions, tt.channelChanged)
			req.NoError(err)
			req.Equal(tt.wantNoSemver, got)

			// semver based
			tt.license.Spec.IsSemverRequired = true
			got, err = getRequiredAirgapUpdates(tt.airgap, tt.license, tt.installedVersions, tt.channelChanged)
			req.NoError(err)
			req.Equal(tt.wantSemver, got)
		})
	}
}
