package airgap

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

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
