package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	storepkg "github.com/replicatedhq/kots/pkg/store"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/replicatedhq/kots/pkg/update/types"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAvailableUpdates(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := mock_store.NewMockStore(ctrl)

	testTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	type args struct {
		kotsStore storepkg.Store
		app       *apptypes.App
		license   *kotsv1beta1.License
	}
	tests := []struct {
		name                      string
		args                      args
		perChannelReleases        map[string][]upstream.ChannelRelease
		setup                     func(t *testing.T, args args, mockServerEndpoint string)
		want                      []types.AvailableUpdate
		wantErr                   bool
		expectedSelectedChannelId string
	}{
		{
			name: "no updates",
			args: args{
				kotsStore: mockStore,
				app: &apptypes.App{
					ID:                "app-id",
					SelectedChannelID: "channel-id",
				},
				license: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						ChannelID:   "channel-id",
						ChannelName: "channel-name",
						AppSlug:     "app-slug",
						LicenseID:   "license-id",
					},
				},
			},
			perChannelReleases: map[string][]upstream.ChannelRelease{},
			setup: func(t *testing.T, args args, licenseEndpoint string) {
				t.Setenv("USE_MOCK_REPORTING", "1")
				args.license.Spec.Endpoint = licenseEndpoint
				mockStore.EXPECT().GetCurrentUpdateCursor(args.app.ID, args.license.Spec.ChannelID).Return("1", nil)
			},
			want:                      []types.AvailableUpdate{},
			wantErr:                   false,
			expectedSelectedChannelId: "channel-id",
		},
		{
			name: "has updates",
			args: args{
				kotsStore: mockStore,
				app: &apptypes.App{
					ID:                "app-id",
					SelectedChannelID: "channel-id",
				},
				license: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						ChannelID:   "channel-id",
						ChannelName: "channel-name",
						AppSlug:     "app-slug",
						LicenseID:   "license-id",
					},
				},
			},
			perChannelReleases: map[string][]upstream.ChannelRelease{
				"channel-id": {
					{
						ChannelSequence: 4,
						ReleaseSequence: 4,
						VersionLabel:    "0.0.4",
						IsRequired:      false,
						CreatedAt:       testTime.Format(time.RFC3339),
						ReleaseNotes:    "release notes",
					},
					{
						ChannelSequence:        3,
						ReleaseSequence:        3,
						VersionLabel:           "0.0.3",
						IsRequired:             false,
						CreatedAt:              testTime.Format(time.RFC3339),
						ReleaseNotes:           "release notes",
						EmbeddedClusterVersion: "2.4.0+k8s-1.32",
					},
					{
						ChannelSequence:        2,
						ReleaseSequence:        2,
						VersionLabel:           "0.0.2",
						IsRequired:             true,
						CreatedAt:              testTime.Format(time.RFC3339),
						ReleaseNotes:           "release notes",
						EmbeddedClusterVersion: "2.4.0+k8s-1.32",
					},
					{
						ChannelSequence:        1,
						ReleaseSequence:        1,
						VersionLabel:           "0.0.1",
						IsRequired:             false,
						CreatedAt:              testTime.Format(time.RFC3339),
						ReleaseNotes:           "release notes",
						EmbeddedClusterVersion: "2.4.0+k8s-1.31",
					},
				},
			},
			setup: func(t *testing.T, args args, licenseEndpoint string) {
				t.Setenv("USE_MOCK_REPORTING", "1")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "2.4.0+k8s-1.30")
				args.license.Spec.Endpoint = licenseEndpoint
				mockStore.EXPECT().GetCurrentUpdateCursor(args.app.ID, args.license.Spec.ChannelID).Return("1", nil)
			},
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "0.0.4",
					UpdateCursor:       "4",
					ChannelID:          "channel-id",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because version 0.0.2 is required and must be deployed first.",
				},
				{
					VersionLabel:       "0.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-id",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because version 0.0.2 is required and must be deployed first.",
				},
				{
					VersionLabel:       "0.0.2",
					UpdateCursor:       "2",
					ChannelID:          "channel-id",
					IsRequired:         true,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       false,
					NonDeployableCause: "Before you can update to this version, you need to update to an earlier version that includes the required infrastructure update.",
				},
				{
					VersionLabel:       "0.0.1",
					UpdateCursor:       "1",
					ChannelID:          "channel-id",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       true,
				},
			},
			wantErr:                   false,
			expectedSelectedChannelId: "channel-id",
		},
		{
			name: "fails to fetch updates",
			args: args{
				kotsStore: mockStore,
				app: &apptypes.App{
					ID:                "app-id",
					SelectedChannelID: "channel-id",
				},
				license: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						ChannelID:   "channel-id",
						ChannelName: "channel-name",
						AppSlug:     "app-slug",
						LicenseID:   "license-id",
					},
				},
			},
			perChannelReleases: map[string][]upstream.ChannelRelease{},
			setup: func(t *testing.T, args args, licenseEndpoint string) {
				t.Setenv("USE_MOCK_REPORTING", "1")
				args.license.Spec.Endpoint = licenseEndpoint
				mockStore.EXPECT().GetCurrentUpdateCursor(args.app.ID, args.license.Spec.ChannelID).Return("1", nil)
			},
			want:                      []types.AvailableUpdate{},
			wantErr:                   true,
			expectedSelectedChannelId: "channel-id",
		},
		{
			name: "uses installed channel id when multi-channel present",
			args: args{
				kotsStore: mockStore,
				app: &apptypes.App{
					ID:                "app-id",
					SelectedChannelID: "channel-id2", // explicitly using the non-default channel
				},
				license: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						ChannelID:   "channel-id",
						ChannelName: "channel-name",
						AppSlug:     "app-slug",
						LicenseID:   "license-id",
						Channels: []kotsv1beta1.Channel{
							{
								ChannelID:   "channel-id",
								ChannelName: "channel-name",
								IsDefault:   true,
							},
							{
								ChannelID:   "channel-id2",
								ChannelName: "channel-name2",
								IsDefault:   false,
							},
						},
					},
				},
			},
			perChannelReleases: map[string][]upstream.ChannelRelease{
				"channel-id": {
					{
						ChannelSequence: 2,
						ReleaseSequence: 2,
						VersionLabel:    "0.0.2",
						IsRequired:      false,
						CreatedAt:       testTime.Format(time.RFC3339),
						ReleaseNotes:    "release notes",
					},
					{
						ChannelSequence: 1,
						ReleaseSequence: 1,
						VersionLabel:    "0.0.1",
						IsRequired:      true,
						CreatedAt:       testTime.Format(time.RFC3339),
						ReleaseNotes:    "release notes",
					},
				},
				"channel-id2": {
					{
						ChannelSequence: 3,
						ReleaseSequence: 3,
						VersionLabel:    "3.0.0",
						IsRequired:      false,
						CreatedAt:       testTime.Format(time.RFC3339),
						ReleaseNotes:    "release notes",
					},
				},
			},
			setup: func(t *testing.T, args args, licenseEndpoint string) {
				t.Setenv("USE_MOCK_REPORTING", "1")
				args.license.Spec.Endpoint = licenseEndpoint
				mockStore.EXPECT().GetCurrentUpdateCursor(args.app.ID, args.license.Spec.Channels[1].ChannelID).Return("1", nil)
			},
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "3.0.0",
					UpdateCursor:       "3",
					ChannelID:          "channel-id2",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       true,
				},
			},
			wantErr:                   false,
			expectedSelectedChannelId: "channel-id2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			mockServer := newMockServerWithReleases(tt.perChannelReleases, tt.expectedSelectedChannelId, tt.wantErr)
			defer mockServer.Close()
			tt.setup(t, tt.args, mockServer.URL)
			got, err := GetAvailableUpdates(tt.args.kotsStore, tt.args.app, tt.args.license)
			if tt.wantErr {
				req.Error(err)
				return
			}
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}

func Test_getAvailableUpdates(t *testing.T) {
	testTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		updates          []upstreamtypes.Update
		currentECVersion string
		want             []types.AvailableUpdate
	}{
		{
			name:             "empty updates",
			updates:          []upstreamtypes.Update{},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want:             []types.AvailableUpdate{},
		},
		{
			name: "single update - deployable",
			updates: []upstreamtypes.Update{
				{
					VersionLabel: "1.0.3",
					Cursor:       "3",
					ChannelID:    "channel-1",
					IsRequired:   false,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
			},
		},
		{
			name: "single required update - deployable",
			updates: []upstreamtypes.Update{
				{
					VersionLabel: "1.0.3",
					Cursor:       "3",
					ChannelID:    "channel-1",
					IsRequired:   true,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         true,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
			},
		},
		{
			name: "multiple updates - no required, all deployable",
			updates: []upstreamtypes.Update{
				{
					VersionLabel: "1.0.4",
					Cursor:       "4",
					ChannelID:    "channel-1",
					IsRequired:   false,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 4",
				},
				{
					VersionLabel: "1.0.3",
					Cursor:       "3",
					ChannelID:    "channel-1",
					IsRequired:   false,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 3",
				},
				{
					VersionLabel: "1.0.2",
					Cursor:       "2",
					ChannelID:    "channel-1",
					IsRequired:   false,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 2",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.4",
					UpdateCursor:       "4",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 4",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 3",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
				{
					VersionLabel:       "1.0.2",
					UpdateCursor:       "2",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 2",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
			},
		},
		{
			name: "one required update blocks later versions",
			updates: []upstreamtypes.Update{
				{
					VersionLabel: "1.0.4",
					Cursor:       "4",
					ChannelID:    "channel-1",
					IsRequired:   false,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 4",
				},
				{
					VersionLabel: "1.0.3",
					Cursor:       "3",
					ChannelID:    "channel-1",
					IsRequired:   false,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 3",
				},
				{
					VersionLabel: "1.0.2",
					Cursor:       "2",
					ChannelID:    "channel-1",
					IsRequired:   true,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 2",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.4",
					UpdateCursor:       "4",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 4",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because version 1.0.2 is required and must be deployed first.",
				},
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 3",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because version 1.0.2 is required and must be deployed first.",
				},
				{
					VersionLabel:       "1.0.2",
					UpdateCursor:       "2",
					ChannelID:          "channel-1",
					IsRequired:         true,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 2",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
			},
		},
		{
			name: "multiple required updates block later versions",
			updates: []upstreamtypes.Update{
				{
					VersionLabel: "1.0.5",
					Cursor:       "5",
					ChannelID:    "channel-1",
					IsRequired:   false,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 5",
				},
				{
					VersionLabel: "1.0.4",
					Cursor:       "4",
					ChannelID:    "channel-1",
					IsRequired:   true,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 4",
				},
				{
					VersionLabel: "1.0.3",
					Cursor:       "3",
					ChannelID:    "channel-1",
					IsRequired:   false,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 3",
				},
				{
					VersionLabel: "1.0.2",
					Cursor:       "2",
					ChannelID:    "channel-1",
					IsRequired:   true,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 2",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.5",
					UpdateCursor:       "5",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 5",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because versions 1.0.4, 1.0.2 are required and must be deployed first.",
				},
				{
					VersionLabel:       "1.0.4",
					UpdateCursor:       "4",
					ChannelID:          "channel-1",
					IsRequired:         true,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 4",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because version 1.0.2 is required and must be deployed first.",
				},
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 3",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because version 1.0.2 is required and must be deployed first.",
				},
				{
					VersionLabel:       "1.0.2",
					UpdateCursor:       "2",
					ChannelID:          "channel-1",
					IsRequired:         true,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 2",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
			},
		},
		{
			name: "kubernetes version incompatible - single update",
			updates: []upstreamtypes.Update{
				{
					VersionLabel:           "1.0.3",
					Cursor:                 "3",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes",
					EmbeddedClusterVersion: "2.5.0+k8s-1.32-rc0",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       false,
					NonDeployableCause: "Before you can update to this version, you need to update to an earlier version that includes the required infrastructure update.",
				},
			},
		},
		{
			name: "kubernetes version incompatible - downgrade",
			updates: []upstreamtypes.Update{
				{
					VersionLabel:           "1.0.3",
					Cursor:                 "3",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes",
					EmbeddedClusterVersion: "2.3.0+k8s-1.29-rc0",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       false,
					NonDeployableCause: "Release includes a downgrade of the infrastructure version, which is not allowed. Cannot use release.",
				},
			},
		},
		{
			name: "kubernetes version incompatible - major version upgrade",
			updates: []upstreamtypes.Update{
				{
					VersionLabel:           "1.0.3",
					Cursor:                 "3",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes",
					EmbeddedClusterVersion: "2.3.0+k8s-2.0",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       false,
					NonDeployableCause: "Release includes a major version upgrade of the infrastructure version, which is not allowed. Cannot use release.",
				},
			},
		},
		{
			name: "kubernetes version compatible",
			updates: []upstreamtypes.Update{
				{
					VersionLabel:           "1.0.3",
					Cursor:                 "3",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes",
					EmbeddedClusterVersion: "2.5.0+k8s-1.31",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
			},
		},
		{
			name: "mixed scenario - k8s compatible and incompatible with most recent deployable",
			updates: []upstreamtypes.Update{
				{
					VersionLabel:           "1.0.4",
					Cursor:                 "4",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes 4",
					EmbeddedClusterVersion: "2.5.0+k8s-1.32",
				},
				{
					VersionLabel:           "1.0.3",
					Cursor:                 "3",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes 3",
					EmbeddedClusterVersion: "2.5.0+k8s-1.31",
				},
				{
					VersionLabel:           "1.0.2",
					Cursor:                 "2",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes 2",
					EmbeddedClusterVersion: "2.5.0+k8s-1.31",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.4",
					UpdateCursor:       "4",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 4",
					IsDeployable:       false,
					NonDeployableCause: "Before you can update to this version, you need to update to an earlier version that includes the required infrastructure update.",
				},
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 3",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
				{
					VersionLabel:       "1.0.2",
					UpdateCursor:       "2",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 2",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
			},
		},
		{
			name: "invalid kubernetes version format",
			updates: []upstreamtypes.Update{
				{
					VersionLabel:           "1.0.3",
					Cursor:                 "3",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes",
					EmbeddedClusterVersion: "2.5.0-invalid-format",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       false,
					NonDeployableCause: "Cannot validate the infrastructure version compatibility for this update. Cannot use release.",
				},
			},
		},
		{
			name: "complex scenario - required updates and k8s compatibility mixed",
			updates: []upstreamtypes.Update{
				{
					VersionLabel:           "1.0.6",
					Cursor:                 "6",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes 6",
					EmbeddedClusterVersion: "2.5.0+k8s-1.32",
				},
				{
					VersionLabel: "1.0.5",
					Cursor:       "5",
					ChannelID:    "channel-1",
					IsRequired:   false,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 5",
				},
				{
					VersionLabel: "1.0.4",
					Cursor:       "4",
					ChannelID:    "channel-1",
					IsRequired:   true,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 4",
				},
				{
					VersionLabel:           "1.0.3",
					Cursor:                 "3",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes 3",
					EmbeddedClusterVersion: "2.5.0+k8s-1.31",
				},
				{
					VersionLabel: "1.0.2",
					Cursor:       "2",
					ChannelID:    "channel-1",
					IsRequired:   true,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 2",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.6",
					UpdateCursor:       "6",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 6",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because versions 1.0.4, 1.0.2 are required and must be deployed first.",
				},
				{
					VersionLabel:       "1.0.5",
					UpdateCursor:       "5",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 5",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because versions 1.0.4, 1.0.2 are required and must be deployed first.",
				},
				{
					VersionLabel:       "1.0.4",
					UpdateCursor:       "4",
					ChannelID:          "channel-1",
					IsRequired:         true,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 4",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because version 1.0.2 is required and must be deployed first.",
				},
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 3",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because version 1.0.2 is required and must be deployed first.",
				},
				{
					VersionLabel:       "1.0.2",
					UpdateCursor:       "2",
					ChannelID:          "channel-1",
					IsRequired:         true,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 2",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
			},
		},
		{
			name: "required updates don't affect k8s checking when no required before current",
			updates: []upstreamtypes.Update{
				{
					VersionLabel:           "1.0.5",
					Cursor:                 "5",
					ChannelID:              "channel-1",
					IsRequired:             true,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes 5",
					EmbeddedClusterVersion: "2.5.0+k8s-1.31",
				},
				{
					VersionLabel:           "1.0.4",
					Cursor:                 "4",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes 4",
					EmbeddedClusterVersion: "2.5.0+k8s-1.32",
				},
				{
					VersionLabel:           "1.0.3",
					Cursor:                 "3",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes 3",
					EmbeddedClusterVersion: "2.5.0+k8s-1.31",
				},
				{
					VersionLabel: "1.0.2",
					Cursor:       "2",
					ChannelID:    "channel-1",
					IsRequired:   false,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 2",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.5",
					UpdateCursor:       "5",
					ChannelID:          "channel-1",
					IsRequired:         true,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 5",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
				{
					VersionLabel:       "1.0.4",
					UpdateCursor:       "4",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 4",
					IsDeployable:       false,
					NonDeployableCause: "Before you can update to this version, you need to update to an earlier version that includes the required infrastructure update.",
				},
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 3",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
				{
					VersionLabel:       "1.0.2",
					UpdateCursor:       "2",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 2",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
			},
		},
		{
			name: "required updates with incompatible k8s versions",
			updates: []upstreamtypes.Update{
				{
					VersionLabel:           "1.0.5",
					Cursor:                 "5",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes 5",
					EmbeddedClusterVersion: "2.5.0+k8s-1.31",
				},
				{
					VersionLabel:           "1.0.4",
					Cursor:                 "4",
					ChannelID:              "channel-1",
					IsRequired:             true,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes 4",
					EmbeddedClusterVersion: "2.5.0+k8s-1.32",
				},
				{
					VersionLabel:           "1.0.3",
					Cursor:                 "3",
					ChannelID:              "channel-1",
					IsRequired:             false,
					ReleasedAt:             &testTime,
					ReleaseNotes:           "release notes 3",
					EmbeddedClusterVersion: "2.5.0+k8s-1.31",
				},
				{
					VersionLabel: "1.0.2",
					Cursor:       "2",
					ChannelID:    "channel-1",
					IsRequired:   false,
					ReleasedAt:   &testTime,
					ReleaseNotes: "release notes 2",
				},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "1.0.5",
					UpdateCursor:       "5",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 5",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because version 1.0.4 is required and must be deployed first.",
				},
				{
					VersionLabel:       "1.0.4",
					UpdateCursor:       "4",
					ChannelID:          "channel-1",
					IsRequired:         true,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 4",
					IsDeployable:       false,
					NonDeployableCause: "Before you can update to this version, you need to update to an earlier version that includes the required infrastructure update.",
				},
				{
					VersionLabel:       "1.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 3",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
				{
					VersionLabel:       "1.0.2",
					UpdateCursor:       "2",
					ChannelID:          "channel-1",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes 2",
					IsDeployable:       true,
					NonDeployableCause: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAvailableUpdates(tt.updates, tt.currentECVersion)
			assert.Equal(t, tt.want, got)
		})
	}
}

func newMockServerWithReleases(preChannelReleases map[string][]upstream.ChannelRelease, expectedSelectedChannelId string, wantErr bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wantErr {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}

		var response struct {
			ChannelReleases []upstream.ChannelRelease `json:"channelReleases"`
		}

		selectedChannelID := r.URL.Query().Get("selectedChannelId")
		if selectedChannelID != expectedSelectedChannelId {
			http.Error(w, "invalid selectedChannelId", http.StatusBadRequest)
			return
		}

		if releases, ok := preChannelReleases[selectedChannelID]; ok {
			response.ChannelReleases = releases
		} else {
			response.ChannelReleases = []upstream.ChannelRelease{}
		}

		w.Header().Set("X-Replicated-UpdateCheckAt", time.Now().Format(time.RFC3339))
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
}
