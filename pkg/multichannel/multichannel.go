package multichannel

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func isSlugInLicenseChannels(slug string, license *kotsv1beta1.License) bool {
	for _, channel := range license.Spec.Channels {
		if channel.ChannelSlug == slug {
			return true
		}
	}
	return false
}

func isMultiChannelLicense(license *kotsv1beta1.License) bool {
	if license == nil {
		return false
	}
	// whether a license is multi-channel is determined by the presence of channels in the license
	// if there are no channels, it is not multi-channel - and was generated before channels
	// were introduced.
	return len(license.Spec.Channels) > 0
}

func canInstallFromChannel(slug string, license *kotsv1beta1.License) bool {
	if !isMultiChannelLicense(license) {
		return true
	}
	return isSlugInLicenseChannels(slug, license)
}

// VerifyAndUpdateLicense will update (if not airgapped), verify that the request channel slug is present, and return the possibly updated license.
// Note that this is a noop if the license passed in is nil.
func VerifyAndUpdateLicense(log *logger.CLILogger, license *kotsv1beta1.License, preferredChannelSlug string, isAirgap bool) (*kotsv1beta1.License, error) {
	if license == nil {
		return nil, nil
	}
	if isAirgap {
		if !canInstallFromChannel(preferredChannelSlug, license) {
			return nil, errors.New("requested channel not found in supplied license")
		}
		return license, nil
	}
	log.ActionWithSpinner("Checking for license update")
	updatedLicense, err := replicatedapp.GetLatestLicense(license)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to get latest license")
	}
	log.FinishSpinner()
	if canInstallFromChannel(preferredChannelSlug, updatedLicense.License) {
		return updatedLicense.License, nil
	}
	return nil, errors.New("requested channel not found in latest license")
}
