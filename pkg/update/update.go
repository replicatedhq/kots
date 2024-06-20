package update

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/reporting"
	storepkg "github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/update/types"
	upstreampkg "github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

// a ephemeral directory to store available updates
var availableUpdatesDir string

func InitAvailableUpdatesDir() error {
	d, err := os.MkdirTemp("", "kotsadm-available-updates")
	if err != nil {
		return errors.Wrap(err, "failed to make temp dir")
	}
	availableUpdatesDir = d
	return nil
}

func GetAvailableUpdatesDir() string {
	return availableUpdatesDir
}

func GetAvailableUpdates(kotsStore storepkg.Store, app *apptypes.App, license *kotsv1beta1.License) ([]types.AvailableUpdate, error) {
	updateCursor, err := kotsStore.GetCurrentUpdateCursor(app.ID, license.Spec.ChannelID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current update cursor")
	}

	upstreamURI := fmt.Sprintf("replicated://%s", license.Spec.AppSlug)
	fetchOptions := &upstreamtypes.FetchOptions{
		License:            license,
		LastUpdateCheckAt:  app.LastUpdateCheckAt,
		CurrentCursor:      updateCursor,
		CurrentChannelID:   license.Spec.ChannelID,
		CurrentChannelName: license.Spec.ChannelName,
		ChannelChanged:     app.ChannelChanged,
		SortOrder:          "desc", // get the latest updates first
		ReportingInfo:      reporting.GetReportingInfo(app.ID),
	}
	updates, err := upstreampkg.GetUpdatesUpstream(upstreamURI, fetchOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get updates")
	}

	availableUpdates := []types.AvailableUpdate{}
	for _, u := range updates.Updates {
		deployable, cause := isUpdateDeployable(u.Cursor, updates.Updates)
		availableUpdates = append(availableUpdates, types.AvailableUpdate{
			VersionLabel:       u.VersionLabel,
			UpdateCursor:       u.Cursor,
			ChannelID:          u.ChannelID,
			IsRequired:         u.IsRequired,
			UpstreamReleasedAt: u.ReleasedAt,
			ReleaseNotes:       u.ReleaseNotes,
			IsDeployable:       deployable,
			NonDeployableCause: cause,
		})
	}

	return availableUpdates, nil
}

func GetAvailableAirgapUpdates(app *apptypes.App, license *kotsv1beta1.License) ([]types.AvailableUpdate, error) {
	updates := []types.AvailableUpdate{}
	if err := filepath.Walk(availableUpdatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		airgap, err := kotsutil.FindAirgapMetaInBundle(path)
		if err != nil {
			return errors.Wrap(err, "failed to find airgap metadata")
		}
		if airgap.Spec.AppSlug != license.Spec.AppSlug {
			return nil
		}
		if airgap.Spec.ChannelID != license.Spec.ChannelID {
			return nil
		}

		deployable, nonDeployableCause, err := IsAirgapUpdateDeployable(app, airgap)
		if err != nil {
			return errors.Wrap(err, "failed to check if airgap update is deployable")
		}

		updates = append(updates, types.AvailableUpdate{
			VersionLabel:       airgap.Spec.VersionLabel,
			UpdateCursor:       airgap.Spec.UpdateCursor,
			ChannelID:          airgap.Spec.ChannelID,
			IsRequired:         airgap.Spec.IsRequired,
			ReleaseNotes:       airgap.Spec.ReleaseNotes,
			IsDeployable:       deployable,
			NonDeployableCause: nonDeployableCause,
		})

		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to walk airgap root dir")
	}

	return updates, nil
}

func RegisterAirgapUpdate(appSlug string, airgapUpdate string) error {
	return RegisterAirgapUpdateInDir(appSlug, airgapUpdate, availableUpdatesDir)
}

func RegisterAirgapUpdateInDir(appSlug string, airgapUpdate string, dir string) error {
	airgap, err := kotsutil.FindAirgapMetaInBundle(airgapUpdate)
	if err != nil {
		return errors.Wrap(err, "failed to find airgap metadata in bundle")
	}
	destPath := filepath.Join(dir, appSlug, fmt.Sprintf("%s-%s.airgap", airgap.Spec.ChannelID, airgap.Spec.UpdateCursor))
	if err := os.MkdirAll(filepath.Dir(destPath), 0744); err != nil {
		return errors.Wrap(err, "failed to create update dir")
	}
	if err := os.Rename(airgapUpdate, destPath); err != nil {
		return errors.Wrap(err, "failed to move airgap update to dest dir")
	}
	return nil
}
