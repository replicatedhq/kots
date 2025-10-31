package license

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
)

func isSlugInLicenseChannels(slug string, license licensewrapper.LicenseWrapper) bool {
	channels := license.GetChannels()
	for _, channel := range channels {
		if channel.ChannelSlug == slug {
			return true
		}
	}
	return false
}

func isMultiChannelLicense(license licensewrapper.LicenseWrapper) bool {
	if !license.IsV1() && !license.IsV2() {
		return false
	}
	// whether a license is multi-channel is determined by the presence of channels in the license
	// if there are no channels, it is not multi-channel - and was generated before channels
	// were introduced.
	channels := license.GetChannels()
	return len(channels) > 0
}

func canInstallFromChannel(slug string, license licensewrapper.LicenseWrapper) bool {
	if !isMultiChannelLicense(license) {
		return true
	}
	return isSlugInLicenseChannels(slug, license)
}

// VerifyAndUpdateLicense will update (if not airgapped), verify that the request channel slug is present, and return the possibly updated license.
// Note that this is a noop if the license passed in is nil.
func VerifyAndUpdateLicense(log *logger.CLILogger, license licensewrapper.LicenseWrapper, preferredChannelSlug string, isAirgap bool) (licensewrapper.LicenseWrapper, error) {
	if !license.IsV1() && !license.IsV2() {
		return licensewrapper.LicenseWrapper{}, nil
	}
	if isAirgap {
		if canInstallFromChannel(preferredChannelSlug, license) {
			return license, nil
		}
		validChannels := []string{}
		channels := license.GetChannels()
		appSlug := license.GetAppSlug()
		for _, channel := range channels {
			validChannels = append(validChannels, fmt.Sprintf("%s/%s", appSlug, channel.ChannelSlug))
		}
		log.Errorf("Channel slug %q is not allowed by license. Please use one of the following: %s", preferredChannelSlug, strings.Join(validChannels, ", "))
		return license, errors.New(fmt.Sprintf("channel slug %q is not allowed by license", preferredChannelSlug))
	}

	log.ActionWithSpinner("Checking for license update")
	// we fetch the latest license to ensure that the license is up to date, before proceeding
	updatedLicense, err := replicatedapp.GetLatestLicense(license, "")
	if err != nil {
		log.FinishSpinnerWithError()
		return licensewrapper.LicenseWrapper{}, errors.Wrap(err, "failed to get latest license")
	}
	log.FinishSpinner()

	if canInstallFromChannel(preferredChannelSlug, updatedLicense.License) {
		return updatedLicense.License, nil
	}
	validChannels := []string{}
	channels := license.GetChannels()
	appSlug := license.GetAppSlug()
	for _, channel := range channels {
		validChannels = append(validChannels, fmt.Sprintf("%s/%s", appSlug, channel.ChannelSlug))
	}
	log.Errorf("Channel slug %q is not allowed by license. Please use one of the following: %s", preferredChannelSlug, strings.Join(validChannels, ", "))
	return updatedLicense.License, errors.New(fmt.Sprintf("channel slug %q is not allowed by latest license", preferredChannelSlug))
}
