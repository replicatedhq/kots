package update

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/cursor"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/update/types"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

// getAvailableUpdates returns the slice of available updates given a list of upstream updates making sure to:
// - Check for previously required versions and setting the deployabled and cause properties accordingly
// - Check for kubernetes version compatibility for embedded cluster versions and setting the deployable and cause properties accordingly
func getAvailableUpdates(updates []upstreamtypes.Update, currentECVersion string) []types.AvailableUpdate {
	availableUpdates := make([]types.AvailableUpdate, len(updates))

	// keep the required updates in a slice in order to add these to the cause properties of each update
	requiredUpdates := []string{}
	// iterate over all updates in reverse since they are sorted in descending order
	for i := len(updates) - 1; i >= 0; i-- {
		upstreamUpdate := updates[i]
		availableUpdates[i] = types.AvailableUpdate{
			VersionLabel:       upstreamUpdate.VersionLabel,
			UpdateCursor:       upstreamUpdate.Cursor,
			ChannelID:          upstreamUpdate.ChannelID,
			IsRequired:         upstreamUpdate.IsRequired,
			UpstreamReleasedAt: upstreamUpdate.ReleasedAt,
			ReleaseNotes:       upstreamUpdate.ReleaseNotes,
			IsDeployable:       true,
		}
		// if there's any required before the current update, mark it as non-deployable and set the cause
		if len(requiredUpdates) > 0 {
			availableUpdates[i].IsDeployable = false
			availableUpdates[i].NonDeployableCause = getRequiredNonDeployableCause(requiredUpdates)
			// else check the k8s versions are compatible but only do so if the update has an embeded cluster version specificied
		} else if upstreamUpdate.EmbeddedClusterVersion != "" {
			if err := util.UpdateWithinKubeRange(currentECVersion, upstreamUpdate.EmbeddedClusterVersion); err != nil {
				availableUpdates[i].IsDeployable = false
				availableUpdates[i].NonDeployableCause = getKubeVersionNonDeployableCause(err)
			}
		}
		// if this update is required add it to the slice so that we can mention it for the next updates
		if upstreamUpdate.IsRequired {
			requiredUpdates = append(requiredUpdates, upstreamUpdate.VersionLabel)
		}
	}

	return availableUpdates
}

func IsAirgapUpdateDeployable(app *apptypes.App, airgap *kotsv1beta1.Airgap) (bool, string, error) {
	appVersions, err := store.GetStore().FindDownstreamVersions(app.ID, true)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get downstream versions")
	}
	license, err := kotsutil.LoadLicenseFromBytes([]byte(app.License))
	if err != nil {
		return false, "", errors.Wrap(err, "failed to load license")
	}
	requiredUpdates, err := getRequiredAirgapUpdates(airgap, license, appVersions.AllVersions, app.ChannelChanged, app.SelectedChannelID)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get missing required versions")
	}
	if len(requiredUpdates) > 0 {
		return false, getRequiredNonDeployableCause(requiredUpdates), nil
	}
	return true, "", nil
}

func getRequiredAirgapUpdates(airgap *kotsv1beta1.Airgap, license *kotsv1beta1.License, installedVersions []*downstreamtypes.DownstreamVersion, channelChanged bool, selectedChannelID string) ([]string, error) {
	requiredUpdates := make([]string, 0)
	// If no versions are installed, we can consider this an initial install.
	// If the channel changed, we can consider this an initial install.
	if len(installedVersions) == 0 || channelChanged {
		return requiredUpdates, nil
	}

	for _, requiredRelease := range airgap.Spec.RequiredReleases {
		laterReleaseInstalled := false
		for _, appVersion := range installedVersions {
			requiredSemver, requiredSemverErr := semver.ParseTolerant(requiredRelease.VersionLabel)

			licenseChan, err := kotsutil.FindChannelInLicense(selectedChannelID, license)
			if err != nil {
				return nil, errors.Wrap(err, "failed to find channel in license during")
			}

			// semvers can be compared across channels
			// if a semmver is missing, fallback to comparing the cursor but only if channel is the same
			if licenseChan.IsSemverRequired && appVersion.Semver != nil && requiredSemverErr == nil {
				if (*appVersion.Semver).GTE(requiredSemver) {
					laterReleaseInstalled = true
					break
				}
			} else {
				// cursors can only be compared on the same channel
				if appVersion.ChannelID != airgap.Spec.ChannelID {
					continue
				}
				if appVersion.Cursor == nil {
					return nil, errors.Errorf("cursor required but version %s does not have cursor", appVersion.UpdateCursor)
				}
				requiredCursor, err := cursor.NewCursor(requiredRelease.UpdateCursor)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse required update cursor %q", requiredRelease.UpdateCursor)
				}
				if (*appVersion.Cursor).After(requiredCursor) || (*appVersion.Cursor).Equal(requiredCursor) {
					laterReleaseInstalled = true
					break
				}
			}
		}

		if !laterReleaseInstalled {
			requiredUpdates = append([]string{requiredRelease.VersionLabel}, requiredUpdates...)
		} else {
			break
		}
	}

	return requiredUpdates, nil
}

// getRequiredNonDeployableCause constructs a non-deployable cause message based on the required updates.
func getRequiredNonDeployableCause(requiredUpdates []string) string {
	if len(requiredUpdates) == 0 {
		return ""
	}
	versionLabels := []string{}
	for _, versionLabel := range requiredUpdates {
		versionLabels = append([]string{versionLabel}, versionLabels...)
	}
	versionLabelsStr := strings.Join(versionLabels, ", ")
	if len(requiredUpdates) == 1 {
		return fmt.Sprintf("This version cannot be deployed because version %s is required and must be deployed first.", versionLabelsStr)
	}
	return fmt.Sprintf("This version cannot be deployed because versions %s are required and must be deployed first.", versionLabelsStr)
}

// getKubeVersionNonDeployableCause constructs a non-deployable cause message based on the kube range validation error message
func getKubeVersionNonDeployableCause(err error) string {
	switch {
	case errors.Is(err, util.ErrKubeMinorRangeMismatch):
		return "Before you can update to this version, you need to update to an earlier version that includes the required infrastructure update."
	case errors.Is(err, util.ErrKubeVersionDowngrade):
		return "Release includes a downgrade of the infrastructure version, which is not allowed. Cannot use release."
	case errors.Is(err, util.ErrKubeMajorVersionUpgrade):
		return "Release includes a major version upgrade of the infrastructure version, which is not allowed. Cannot use release."
	}
	return "Cannot validate the infrastructure version compatibility for this update. Cannot use release."
}
