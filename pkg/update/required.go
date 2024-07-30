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
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func isUpdateDeployable(updateCursor string, updates []upstreamtypes.Update) (bool, string) {
	// iterate over updates in reverse since they are sorted in descending order
	requiredUpdates := []string{}
	for i := len(updates) - 1; i >= 0; i-- {
		if updates[i].Cursor == updateCursor {
			break
		}
		if updates[i].IsRequired {
			requiredUpdates = append(requiredUpdates, updates[i].VersionLabel)
		}
	}
	if len(requiredUpdates) > 0 {
		return false, getNonDeployableCause(requiredUpdates)
	}
	return true, ""
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
		return false, getNonDeployableCause(requiredUpdates), nil
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
				return nil, errors.Wrap(err, "failed to find channel in license")
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

func getNonDeployableCause(requiredUpdates []string) string {
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
