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
						ChannelSequence:        3,
						ReleaseSequence:        3,
						VersionLabel:           "0.0.3",
						IsRequired:             false,
						CreatedAt:              testTime.Format(time.RFC3339),
						ReleaseNotes:           "release notes",
						EmbeddedClusterVersion: "2.4.0+k8s-1.33",
					},
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
			},
			setup: func(t *testing.T, args args, licenseEndpoint string) {
				t.Setenv("USE_MOCK_REPORTING", "1")
				t.Setenv("EMBEDDED_CLUSTER_VERSION", "2.4.0+k8s-1.30")
				args.license.Spec.Endpoint = licenseEndpoint
				mockStore.EXPECT().GetCurrentUpdateCursor(args.app.ID, args.license.Spec.ChannelID).Return("1", nil)
			},
			want: []types.AvailableUpdate{
				{
					VersionLabel:       "0.0.3",
					UpdateCursor:       "3",
					ChannelID:          "channel-id",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because of incompatible kubernetes versions, cannot update by more than one kubernetes minor version: current 1.30.0, update 1.33.0",
				},
				{
					VersionLabel:       "0.0.2",
					UpdateCursor:       "2",
					ChannelID:          "channel-id",
					IsRequired:         false,
					UpstreamReleasedAt: &testTime,
					ReleaseNotes:       "release notes",
					IsDeployable:       false,
					NonDeployableCause: "This version cannot be deployed because version 0.0.1 is required and must be deployed first.",
				},
				{
					VersionLabel:       "0.0.1",
					UpdateCursor:       "1",
					ChannelID:          "channel-id",
					IsRequired:         true,
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
func TestIsUpdateDeployable(t *testing.T) {
	tests := []struct {
		name             string
		updateIndex      int
		updates          []upstreamtypes.Update
		currentECVersion string
		want             bool
		wantCause        string
	}{
		{
			name:        "one update",
			updateIndex: 0,
			updates: []upstreamtypes.Update{
				{VersionLabel: "1.0.3", Cursor: "3", IsRequired: false},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want:             true,
			wantCause:        "",
		},
		{
			name:        "no required updates",
			updateIndex: 1,
			updates: []upstreamtypes.Update{
				{VersionLabel: "1.0.4", Cursor: "4", IsRequired: false},
				{VersionLabel: "1.0.3", Cursor: "3", IsRequired: false},
				{VersionLabel: "1.0.2", Cursor: "2", IsRequired: false},
				{VersionLabel: "1.0.1", Cursor: "1", IsRequired: false},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want:             true,
			wantCause:        "",
		},
		{
			name:        "no required updates and embedded cluster kubernetes version within range",
			updateIndex: 1,
			updates: []upstreamtypes.Update{
				{VersionLabel: "1.0.4", Cursor: "4", IsRequired: false},
				{VersionLabel: "1.0.3", Cursor: "3", IsRequired: false, EmbeddedClusterVersion: "2.5.0+k8s-1.31"},
				{VersionLabel: "1.0.2", Cursor: "2", IsRequired: false},
				{VersionLabel: "1.0.1", Cursor: "1", IsRequired: false},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want:             true,
			wantCause:        "",
		},
		{
			name:        "no required releases before it and embedded cluster kubernetes version within range",
			updateIndex: 1,
			updates: []upstreamtypes.Update{
				{VersionLabel: "1.0.4", Cursor: "4", IsRequired: true},
				{VersionLabel: "1.0.3", Cursor: "3", IsRequired: false, EmbeddedClusterVersion: "2.5.0+k8s-1.31"},
				{VersionLabel: "1.0.2", Cursor: "2", IsRequired: false},
				{VersionLabel: "1.0.1", Cursor: "1", IsRequired: false},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want:             true,
			wantCause:        "",
		},
		{
			name:        "one required release before it",
			updateIndex: 1,
			updates: []upstreamtypes.Update{
				{VersionLabel: "1.0.4", Cursor: "4", IsRequired: false},
				{VersionLabel: "1.0.3", Cursor: "3", IsRequired: false},
				{VersionLabel: "1.0.2", Cursor: "2", IsRequired: true},
				{VersionLabel: "1.0.1", Cursor: "1", IsRequired: false},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want:             false,
			wantCause:        "This version cannot be deployed because version 1.0.2 is required and must be deployed first.",
		},
		{
			name:        "two required releases before it",
			updateIndex: 1,
			updates: []upstreamtypes.Update{
				{VersionLabel: "1.0.4", Cursor: "4", IsRequired: false},
				{VersionLabel: "1.0.3", Cursor: "3", IsRequired: false},
				{VersionLabel: "1.0.2", Cursor: "2", IsRequired: true},
				{VersionLabel: "1.0.1", Cursor: "1", IsRequired: true},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want:             false,
			wantCause:        "This version cannot be deployed because versions 1.0.2, 1.0.1 are required and must be deployed first.",
		},
		{
			name:        "incompatible kubernetes versions on embedded cluster version",
			updateIndex: 1,
			updates: []upstreamtypes.Update{
				{VersionLabel: "1.0.4", Cursor: "4", IsRequired: false},
				{VersionLabel: "1.0.3", Cursor: "3", IsRequired: false, EmbeddedClusterVersion: "2.5.0+k8s-1.32-rc0"},
				{VersionLabel: "1.0.2", Cursor: "2", IsRequired: false},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want:             false,
			wantCause:        "This version cannot be deployed because of incompatible kubernetes versions, cannot update by more than one kubernetes minor version: current 1.30.0, update 1.32.0",
		},
		{
			name:        "incompatible kubernetes versions on embedded cluster version (downgrade)",
			updateIndex: 0,
			updates: []upstreamtypes.Update{
				{VersionLabel: "1.0.3", Cursor: "3", IsRequired: false, EmbeddedClusterVersion: "2.3.0+k8s-1.29-rc0"},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want:             false,
			wantCause:        "This version cannot be deployed because of incompatible kubernetes versions, cannot downgrade the kubernetes version: current 1.30.0, update 1.29.0",
		},
		{
			name:        "incompatible kubernetes versions on embedded cluster version and one required release before it",
			updateIndex: 1,
			updates: []upstreamtypes.Update{
				{VersionLabel: "1.0.4", Cursor: "4", IsRequired: false},
				{VersionLabel: "1.0.3", Cursor: "3", IsRequired: false, EmbeddedClusterVersion: "2.5.0+k8s-1.32-rc0"},
				{VersionLabel: "1.0.2", Cursor: "2", IsRequired: true},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want:             false,
			wantCause:        "This version cannot be deployed because of incompatible kubernetes versions, cannot update by more than one kubernetes minor version: current 1.30.0, update 1.32.0",
		},
		{
			name:        "invalid embedded cluster update version format",
			updateIndex: 0,
			updates: []upstreamtypes.Update{
				{VersionLabel: "1.0.3", Cursor: "3", IsRequired: false, EmbeddedClusterVersion: "2.5.0-invalid-format"},
			},
			currentECVersion: "2.4.0+k8s-1.30-rc0",
			want:             false,
			wantCause:        "This version cannot be deployed because of incompatible kubernetes versions, failed to extract update kube version: failed to extract kube version from '2.5.0-invalid-format'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, msg := isUpdateDeployable(tt.updates[tt.updateIndex], tt.updates, tt.currentECVersion)
			assert.Equal(t, tt.want, result)
			assert.Equal(t, tt.wantCause, msg)
		})
	}
}
