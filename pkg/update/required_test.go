package update

import (
	"testing"

	"github.com/blang/semver"
	"github.com/golang/mock/gomock"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/store"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/stretchr/testify/require"
)

func Test_getRequiredAirgapUpdates(t *testing.T) {
	channelID := "channel-id"
	channelName := "channel-name"

	testLicense := &kotsv1beta1.License{
		Spec: kotsv1beta1.LicenseSpec{
			ChannelID:   "default-channel-id",
			ChannelName: "Default Channel",
			Channels: []kotsv1beta1.Channel{
				{
					ChannelID:        "default-channel-id",
					ChannelName:      "Default Channel",
					IsDefault:        true,
					IsSemverRequired: true,
				},
				{
					ChannelID:        channelID,
					ChannelName:      channelName,
					IsDefault:        false,
					IsSemverRequired: true,
				},
			},
		},
	}

	tests := []struct {
		name              string
		airgap            *kotsv1beta1.Airgap
		license           *kotsv1beta1.License
		installedVersions []*downstreamtypes.DownstreamVersion
		channelChanged    bool
		wantSemver        []string
		wantNoSemver      []string
		selectedChannelID string
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
			selectedChannelID: channelID,
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
			license: testLicense,
			installedVersions: []*downstreamtypes.DownstreamVersion{
				{
					ChannelID:    channelID,
					VersionLabel: "0.1.124",
					UpdateCursor: "124",
				},
			},
			wantNoSemver:      []string{},
			wantSemver:        []string{},
			selectedChannelID: channelID,
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
			license: testLicense,
			installedVersions: []*downstreamtypes.DownstreamVersion{
				{
					ChannelID:    channelID,
					VersionLabel: "0.1.117",
					UpdateCursor: "117",
				},
			},
			wantNoSemver:      []string{"0.1.120", "0.1.123"},
			wantSemver:        []string{"0.1.120", "0.1.123"},
			selectedChannelID: channelID,
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
			license: testLicense,
			installedVersions: []*downstreamtypes.DownstreamVersion{
				{
					ChannelID:    channelID,
					VersionLabel: "0.1.113",
					UpdateCursor: "113",
				},
			},
			wantNoSemver:      []string{"0.1.115", "0.1.120", "0.1.123"},
			wantSemver:        []string{"0.1.115", "0.1.120", "0.1.123"},
			selectedChannelID: channelID,
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
			license:        testLicense,
			channelChanged: true,
			installedVersions: []*downstreamtypes.DownstreamVersion{
				{
					ChannelID:    "different-channel",
					VersionLabel: "0.1.117",
					UpdateCursor: "117",
				},
			},
			wantNoSemver:      []string{},
			wantSemver:        []string{},
			selectedChannelID: channelID,
		},
		{
			name: "check across multiple channels with multi chan license",
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
				Spec: kotsv1beta1.LicenseSpec{
					ChannelID:   "stable-channel", // intentionally fully avoiding the default channel
					ChannelName: "Stable Channel",
					Channels: []kotsv1beta1.Channel{
						{
							ChannelID:        "stable-channel",
							ChannelName:      "Stable Channel",
							ChannelSlug:      "stable-channel",
							IsDefault:        false,
							IsSemverRequired: true,
						},
						{
							ChannelID:        "different-channel",
							ChannelName:      "Different Channel",
							ChannelSlug:      "different-channel",
							IsDefault:        true,
							IsSemverRequired: false,
						},
						{
							ChannelID:        channelID,
							ChannelName:      channelName,
							ChannelSlug:      channelID,
							IsDefault:        false,
							IsSemverRequired: true,
						},
					},
				},
			},
			channelChanged: true,
			installedVersions: []*downstreamtypes.DownstreamVersion{
				{
					ChannelID:    "different-channel",
					VersionLabel: "0.1.117",
					UpdateCursor: "117",
				},
			},
			wantNoSemver:      []string{},
			wantSemver:        []string{},
			selectedChannelID: channelID,
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
			licenseWrapper := licensewrapper.LicenseWrapper{V1: tt.license}
			got, err := getRequiredAirgapUpdates(tt.airgap, &licenseWrapper, tt.installedVersions, tt.channelChanged, tt.selectedChannelID)
			req.NoError(err)
			req.Equal(tt.wantNoSemver, got)

			// semver based
			tt.license.Spec.IsSemverRequired = true
			licenseWrapper = licensewrapper.LicenseWrapper{V1: tt.license}
			got, err = getRequiredAirgapUpdates(tt.airgap, &licenseWrapper, tt.installedVersions, tt.channelChanged, tt.selectedChannelID)
			req.NoError(err)
			req.Equal(tt.wantSemver, got)
		})
	}
}

func TestIsAirgapUpdateDeployable(t *testing.T) {
	tests := []struct {
		name                string
		airgap              *kotsv1beta1.Airgap
		currentECVersion    string
		wantDeployable      bool
		wantCause           string
		isV3EmbeddedCluster bool
		setupStore          func(mockStore *mock_store.MockStore)
	}{
		{
			name: "not deployable due to required releases",
			airgap: &kotsv1beta1.Airgap{
				Spec: kotsv1beta1.AirgapSpec{
					ChannelID: "test-channel",
					RequiredReleases: []kotsv1beta1.AirgapReleaseMeta{
						{
							VersionLabel: "1.0.0",
							UpdateCursor: "100",
						},
					},
					AirgapReleaseMeta: kotsv1beta1.AirgapReleaseMeta{
						VersionLabel:           "1.1.0",
						UpdateCursor:           "110",
						EmbeddedClusterVersion: "2.7.4+k8s-1.30",
					},
				},
			},
			currentECVersion: "2.7.3+k8s-1.29",
			wantDeployable:   false,
			wantCause:        "This version cannot be deployed because version 1.0.0 is required and must be deployed first.",
			setupStore: func(mockStore *mock_store.MockStore) {
				s := semver.MustParse("0.9.0")
				c := cursor.MustParse("90")
				mockStore.EXPECT().FindDownstreamVersions("test-app", true).Return(&downstreamtypes.DownstreamVersions{
					AllVersions: []*downstreamtypes.DownstreamVersion{
						{
							ChannelID:    "test-channel",
							VersionLabel: "0.9.0",
							UpdateCursor: "90",
							Semver:       &s,
							Cursor:       &c,
						},
					},
				}, nil)
			},
		},
		{
			name: "deployable when no required updates and compatible EC version",
			airgap: &kotsv1beta1.Airgap{
				Spec: kotsv1beta1.AirgapSpec{
					ChannelID: "test-channel",
					AirgapReleaseMeta: kotsv1beta1.AirgapReleaseMeta{
						EmbeddedClusterVersion: "2.7.4+k8s-1.30",
					},
				},
			},
			currentECVersion: "2.7.3+k8s-1.29",
			wantDeployable:   true,
			wantCause:        "",
			setupStore: func(mockStore *mock_store.MockStore) {
				mockStore.EXPECT().FindDownstreamVersions("test-app", true).Return(&downstreamtypes.DownstreamVersions{
					AllVersions: []*downstreamtypes.DownstreamVersion{},
				}, nil)
			},
		},
		{
			name: "not deployable due to EC version incompatibility",
			airgap: &kotsv1beta1.Airgap{
				Spec: kotsv1beta1.AirgapSpec{
					ChannelID: "test-channel",
					AirgapReleaseMeta: kotsv1beta1.AirgapReleaseMeta{
						EmbeddedClusterVersion: "2.8.0+k8s-1.31",
					},
				},
			},
			currentECVersion: "2.7.4+k8s-1.29",
			wantDeployable:   false,
			wantCause:        "Before you can update to this version, you need to update to an earlier version that includes the required infrastructure update.",
			setupStore: func(mockStore *mock_store.MockStore) {
				mockStore.EXPECT().FindDownstreamVersions("test-app", true).Return(&downstreamtypes.DownstreamVersions{
					AllVersions: []*downstreamtypes.DownstreamVersion{},
				}, nil)
			},
		},
		{
			name: "deployable when airgap EC version is empty",
			airgap: &kotsv1beta1.Airgap{
				Spec: kotsv1beta1.AirgapSpec{
					ChannelID: "test-channel",
					AirgapReleaseMeta: kotsv1beta1.AirgapReleaseMeta{
						EmbeddedClusterVersion: "",
					},
				},
			},
			currentECVersion: "2.7.3+k8s-1.29",
			wantDeployable:   true,
			wantCause:        "",
			setupStore: func(mockStore *mock_store.MockStore) {
				mockStore.EXPECT().FindDownstreamVersions("test-app", true).Return(&downstreamtypes.DownstreamVersions{
					AllVersions: []*downstreamtypes.DownstreamVersion{},
				}, nil)
			},
		},
		{
			name: "deployable when current EC version is empty (non-ec installation) even if airgap EC version is set",
			airgap: &kotsv1beta1.Airgap{
				Spec: kotsv1beta1.AirgapSpec{
					ChannelID: "test-channel",
					AirgapReleaseMeta: kotsv1beta1.AirgapReleaseMeta{
						EmbeddedClusterVersion: "2.7.5+k8s-1.30",
					},
				},
			},
			currentECVersion: "",
			wantDeployable:   true,
			wantCause:        "",
			setupStore: func(mockStore *mock_store.MockStore) {
				mockStore.EXPECT().FindDownstreamVersions("test-app", true).Return(&downstreamtypes.DownstreamVersions{
					AllVersions: []*downstreamtypes.DownstreamVersion{},
				}, nil)
			},
		},
		{
			name: "always deployable in V3 Embedded Cluster (version checks handled separately)",
			airgap: &kotsv1beta1.Airgap{
				Spec: kotsv1beta1.AirgapSpec{
					ChannelID: "test-channel",
					RequiredReleases: []kotsv1beta1.AirgapReleaseMeta{
						{
							VersionLabel: "1.0.0",
							UpdateCursor: "100",
						},
					},
					AirgapReleaseMeta: kotsv1beta1.AirgapReleaseMeta{
						VersionLabel:           "1.1.0",
						UpdateCursor:           "110",
						EmbeddedClusterVersion: "2.8.0+k8s-1.31",
					},
				},
			},
			currentECVersion:    "2.7.3+k8s-1.29",
			wantDeployable:      true,
			wantCause:           "",
			isV3EmbeddedCluster: true,
			setupStore: func(mockStore *mock_store.MockStore) {
				// No store calls should be made in V3 EC mode
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Set V3 EC environment
			if tt.isV3EmbeddedCluster {
				t.Setenv("IS_EMBEDDED_CLUSTER_V3", "true")
			}

			mockStore := mock_store.NewMockStore(ctrl)
			tt.setupStore(mockStore)

			store.SetStore(mockStore)
			defer store.SetStore(nil)

			app := &apptypes.App{
				ID:                "test-app",
				SelectedChannelID: "test-channel",
				License: `apiVersion: kots.io/v1beta1
kind: License
spec:
  channelID: test-channel
  channels:
  - channelID: test-channel
    channelName: Test
    isDefault: true
    isSemverRequired: false`,
			}

			deployable, cause, err := IsAirgapUpdateDeployable(app, tt.airgap, tt.currentECVersion)

			require.NoError(t, err)
			require.Equal(t, tt.wantDeployable, deployable)
			require.Equal(t, tt.wantCause, cause)
		})
	}
}
