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
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/update/types"
	upstreampkg "github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
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

func GetAvailableUpdates(kotsStore storepkg.Store, app *apptypes.App, license *licensewrapper.LicenseWrapper) ([]types.AvailableUpdate, error) {
	licenseChan, err := kotsutil.FindChannelInLicense(app.SelectedChannelID, license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find channel in license")
	}

	updateCursor, err := kotsStore.GetCurrentUpdateCursor(app.ID, licenseChan.ChannelID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current update cursor")
	}

	upstreamURI := fmt.Sprintf("replicated://%s", license.GetAppSlug())
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

	currentECVersion := util.EmbeddedClusterVersion()

	availableUpdates := getAvailableUpdates(updates.Updates, currentECVersion)

	// additional deployable checks against current version
	downstreams, err := kotsStore.ListDownstreamsForApp(app.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}
	if len(downstreams) == 0 {
		return availableUpdates, nil
	}

	currentVersion, err := kotsStore.GetCurrentDownstreamVersion(app.ID, downstreams[0].ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current downstream version")
	}
	if currentVersion != nil &&
		currentVersion.Status == storetypes.VersionFailed &&
		currentVersion.IsRequired {
		// none of the upstream available updates are deployable if current version is required and failed to deploy
		for i := range availableUpdates {
			availableUpdates[i].IsDeployable = false
			availableUpdates[i].NonDeployableCause = fmt.Sprintf("Cannot deploy this version because required version %s failed to deploy. Please retry deploying version %s or check for new updates.", currentVersion.VersionLabel, currentVersion.VersionLabel)
		}
	}

	return availableUpdates, nil
}

func GetAvailableAirgapUpdates(app *apptypes.App, license *licensewrapper.LicenseWrapper) ([]types.AvailableUpdate, error) {
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
		if airgap.Spec.AppSlug != license.GetAppSlug() {
			return nil
		}
		if _, err = kotsutil.FindChannelInLicense(airgap.Spec.ChannelID, license); err != nil {
			logger.Info("skipping airgap update check for channel not found in current license",
				zap.String("airgap_channelName", airgap.Spec.ChannelName),
				zap.String("airgap_channelID", airgap.Spec.ChannelID),
			)
			return nil // skip airgap updates that are not for the current channel, preserving previous behavior
		}

		currentECVersion := util.EmbeddedClusterVersion()
		deployable, nonDeployableCause, err := IsAirgapUpdateDeployable(app, airgap, currentECVersion)
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
