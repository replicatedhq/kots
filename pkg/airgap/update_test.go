package airgap

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/require"
)

func Test_canInstall(t *testing.T) {
	type args struct {
		beforeKotsKinds *kotsutil.KotsKinds
		afterKotsKinds  *kotsutil.KotsKinds
		license         *licensewrapper.LicenseWrapper
	}
	tests := []struct {
		name                string
		args                args
		wantErr             bool
		isV3EmbeddedCluster bool
	}{
		{
			name: "semver not enabled, version labels are dfferent, and cursors are different",
			args: args{
				beforeKotsKinds: &kotsutil.KotsKinds{
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
				license: &licensewrapper.LicenseWrapper{
					V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: false,
					},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "semver not enabled, version labels match, and cursors are different",
			args: args{
				beforeKotsKinds: &kotsutil.KotsKinds{
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
				license: &licensewrapper.LicenseWrapper{
					V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: false,
					},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "semver not enabled, version labels match, and cursors match",
			args: args{
				beforeKotsKinds: &kotsutil.KotsKinds{
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
				license: &licensewrapper.LicenseWrapper{
					V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: false,
					},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "semver enabled, version labels are dfferent, and cursors are different",
			args: args{
				beforeKotsKinds: &kotsutil.KotsKinds{
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
				license: &licensewrapper.LicenseWrapper{
					V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: true,
					},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "semver enabled, version labels match, and cursors are different",
			args: args{
				beforeKotsKinds: &kotsutil.KotsKinds{
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
				license: &licensewrapper.LicenseWrapper{
					V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: true,
					},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "V3 Embedded Cluster - always allows installation (version checks handled separately)",
			args: args{
				beforeKotsKinds: &kotsutil.KotsKinds{
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
					License: &licensewrapper.LicenseWrapper{
						V1: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							ChannelID: "test-channel-id",
						},
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
				license: &licensewrapper.LicenseWrapper{
					V1: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						IsSemverRequired: true,
					},
					},
				},
			},
			wantErr:             false, // Should not error in V3 EC mode even with same cursor/version
			isV3EmbeddedCluster: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// Set V3 EC environment
			if tt.isV3EmbeddedCluster {
				t.Setenv("IS_EMBEDDED_CLUSTER_V3", "true")
			}

			err := canInstall(tt.args.beforeKotsKinds, tt.args.afterKotsKinds, tt.args.license)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
		})
	}
}
