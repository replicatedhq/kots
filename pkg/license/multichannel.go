package license

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

// getDefaultChannelSlug returns the channel slug of the default channel in the license.
// If passed a pre multi-channel license, it will return an empty string as pre multi-channel
// licenses did not store a channel slug.
func getDefaultChannelSlug(license *kotsv1beta1.License) (string, error) {
	if !isMultiChannelLicense(license) {
		return "", nil
	}

	if len(license.Spec.Channels) == 1 {
		return license.Spec.Channels[0].ChannelSlug, nil
	}

	for _, channel := range license.Spec.Channels {
		if channel.IsDefault {
			return channel.ChannelSlug, nil
		}
	}

	//should never happen as there should always be a default channel in a multi-channel license - even if it is the only channel
	return "", errors.New("no default channel slug found in license")
}

// VerifyAndUpdateLicense will update (if not airgapped) the license, verify that the requested channel slug is present if one is supplied, and return the latest
// instance of the license and the associated channel slug (using the default if it was used).
func VerifyAndUpdateLicense(log *logger.CLILogger, license *kotsv1beta1.License, preferredChannelSlug string, isAirgap bool) (string, *kotsv1beta1.License, error) {
	if license == nil {
		return preferredChannelSlug, nil, nil
	}

	if isAirgap {
		if isMultiChannelLicense(license) && preferredChannelSlug == "" { // we'll be installing the default channel
			defaultChannelSlug, err := getDefaultChannelSlug(license)
			if err != nil {
				return "", nil, errors.Wrap(err, "failed to find default channel slug and no explicit channel slug was provided")
			}
			return defaultChannelSlug, license, nil
		}

		if canInstallFromChannel(preferredChannelSlug, license) {
			return preferredChannelSlug, license, nil
		}
		return preferredChannelSlug, nil, errors.New("requested channel not found in license")
	}

	log.ActionWithSpinner("Checking for license update")
	// we fetch the latest license to ensure that the license is up to date, before proceeding
	updatedLicense, err := replicatedapp.GetLatestLicense(license, "")
	if err != nil {
		log.FinishSpinnerWithError()
	}
	log.FinishSpinner()

	if preferredChannelSlug == "" { // we'll be installing the default channel
		defaultChannelSlug, err := getDefaultChannelSlug(updatedLicense.License)
		if err != nil {
			return "", nil, errors.Wrap(err, "failed to find default channel slug and no explicit channel slug was provided")
		}
		return defaultChannelSlug, updatedLicense.License, nil
	}

	if canInstallFromChannel(preferredChannelSlug, updatedLicense.License) {
		return preferredChannelSlug, updatedLicense.License, nil
	}
	return preferredChannelSlug, nil, errors.New("requested channel not found in latest license")
}
