package update

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/reporting"
	storepkg "github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/update/types"
	upstreampkg "github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"go.uber.org/zap"
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

func GetAvailableUpdates(kotsStore storepkg.Store, app *apptypes.App, license *kotsv1beta1.License) ([]types.AvailableUpdate, error) {
	licenseChan, err := kotsutil.FindChannelInLicense(app.SelectedChannelID, license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find channel in license")
	}

	updateCursor, err := kotsStore.GetCurrentUpdateCursor(app.ID, licenseChan.ChannelID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current update cursor")
	}

	upstreamURI := fmt.Sprintf("replicated://%s", license.Spec.AppSlug)
	fetchOptions := &upstreamtypes.FetchOptions{
		License:            license,
		LastUpdateCheckAt:  app.LastUpdateCheckAt,
		CurrentCursor:      updateCursor,
		CurrentChannelID:   licenseChan.ChannelID,
		CurrentChannelName: licenseChan.ChannelName,
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
		if filepath.Ext(path) != ".airgap" {
			return nil
		}

		airgap, err := kotsutil.FindAirgapMetaInBundle(path)
		if err != nil {
			return errors.Wrap(err, "failed to find airgap metadata")
		}
		if airgap.Spec.AppSlug != license.Spec.AppSlug {
			return nil
		}
		if _, err = kotsutil.FindChannelInLicense(airgap.Spec.ChannelID, license); err != nil {
			logger.Info("skipping airgap update check for channel not found in current license",
				zap.String("airgap_channelName", airgap.Spec.ChannelName),
				zap.String("airgap_channelID", airgap.Spec.ChannelID),
			)
			return nil // skip airgap updates that are not for the current channel, preserving previous behavior
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
	airgap, err := kotsutil.FindAirgapMetaInBundle(airgapUpdate)
	if err != nil {
		return errors.Wrap(err, "failed to find airgap metadata in bundle")
	}
	destPath := getAirgapUpdatePath(appSlug, airgap.Spec.ChannelID, airgap.Spec.UpdateCursor)
	if err := os.MkdirAll(filepath.Dir(destPath), 0744); err != nil {
		return errors.Wrap(err, "failed to create update dir")
	}
	if err := os.Rename(airgapUpdate, destPath); err != nil {
		return errors.Wrap(err, "failed to move airgap update to dest dir")
	}
	return nil
}

func RemoveAirgapUpdate(appSlug string, channelID string, updateCursor string) error {
	updatePath := getAirgapUpdatePath(appSlug, channelID, updateCursor)
	if err := os.Remove(updatePath); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to remove")
	}
	return nil
}

func GetAirgapUpdate(appSlug string, channelID string, updateCursor string) (string, error) {
	updatePath := getAirgapUpdatePath(appSlug, channelID, updateCursor)
	if _, err := os.Stat(updatePath); err != nil {
		return "", errors.Wrap(err, "failed to stat")
	}
	return updatePath, nil
}

func getAirgapUpdatePath(appSlug string, channelID string, updateCursor string) string {
	return filepath.Join(availableUpdatesDir, appSlug, fmt.Sprintf("%s-%s.airgap", channelID, updateCursor))
}
