package airgap

import (
	"testing"

	"github.com/blang/semver"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

func Test_getMissingRequiredVersions(t *testing.T) {
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
			got, err := getMissingRequiredVersions(tt.airgap, tt.license, tt.installedVersions, tt.channelChanged)
			req.NoError(err)
			req.Equal(tt.wantNoSemver, got)

			// semver based
			tt.license.Spec.IsSemverRequired = true
			got, err = getMissingRequiredVersions(tt.airgap, tt.license, tt.installedVersions, tt.channelChanged)
			req.NoError(err)
			req.Equal(tt.wantSemver, got)
		})
	}
}

func Test_canInstall(t *testing.T) {
	type args struct {
		beforeKotsKinds *kotsutil.KotsKinds
		afterKotsKinds  *kotsutil.KotsKinds
		license         *kotsv1beta1.License
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "semver not enabled, version labels are dfferent, and cursors are different",
			args: args{
				beforeKotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							ChannelID:    "test-channel-id",
							UpdateCursor: "1",
							VersionLabel: "0.1.1",
						},
					},
				},
				afterKotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							ChannelID:    "test-channel-id",
							UpdateCursor: "2",
							VersionLabel: "0.1.2",
						},
					},
				},
				license: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "semver not enabled, version labels match, and cursors are different",
			args: args{
				beforeKotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							ChannelID:    "test-channel-id",
							UpdateCursor: "1",
							VersionLabel: "0.1.1",
						},
					},
				},
				afterKotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							ChannelID:    "test-channel-id",
							UpdateCursor: "2",
							VersionLabel: "0.1.1",
						},
					},
				},
				license: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "semver not enabled, version labels match, and cursors match",
			args: args{
				beforeKotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							ChannelID:    "test-channel-id",
							UpdateCursor: "1",
							VersionLabel: "0.1.1",
						},
					},
				},
				afterKotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							ChannelID:    "test-channel-id",
							UpdateCursor: "1",
							VersionLabel: "0.1.1",
						},
					},
				},
				license: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: false,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "semver enabled, version labels are dfferent, and cursors are different",
			args: args{
				beforeKotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							ChannelID:    "test-channel-id",
							UpdateCursor: "1",
							VersionLabel: "0.1.1",
						},
					},
				},
				afterKotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							ChannelID:    "test-channel-id",
							UpdateCursor: "2",
							VersionLabel: "0.1.2",
						},
					},
				},
				license: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "semver enabled, version labels match, and cursors are different",
			args: args{
				beforeKotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							ChannelID:    "test-channel-id",
							UpdateCursor: "1",
							VersionLabel: "0.1.1",
						},
					},
				},
				afterKotsKinds: &kotsutil.KotsKinds{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
					},
					Installation: kotsv1beta1.Installation{
						Spec: kotsv1beta1.InstallationSpec{
							ChannelID:    "test-channel-id",
							UpdateCursor: "2",
							VersionLabel: "0.1.1",
						},
					},
				},
				license: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: true,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			err := canInstall(tt.args.beforeKotsKinds, tt.args.afterKotsKinds, tt.args.license)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
		})
	}
}
