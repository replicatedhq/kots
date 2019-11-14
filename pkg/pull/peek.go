package pull

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upstream"
)

type GetUpdatesOptions struct {
	HelmRepoURI   string
	Namespace     string
	LocalPath     string
	LicenseFile   string
	CurrentCursor string
	Silent        bool
}

// GetUpdates will retrieve all later versions of the application specified in upstreamURI
// using the options specified in getUpdatesOptions. It returns a list of versions.
func GetUpdates(upstreamURI string, getUpdatesOptions GetUpdatesOptions) ([]upstream.Update, error) {
	log := logger.NewLogger()

	if getUpdatesOptions.Silent {
		log.Silence()
	}

	log.Initialize()

	fetchOptions := upstream.FetchOptions{}
	fetchOptions.HelmRepoURI = getUpdatesOptions.HelmRepoURI
	fetchOptions.LocalPath = getUpdatesOptions.LocalPath
	fetchOptions.CurrentCursor = getUpdatesOptions.CurrentCursor

	if getUpdatesOptions.LicenseFile != "" {
		license, err := parseLicenseFromFile(getUpdatesOptions.LicenseFile)
		if err != nil {
			if errors.Cause(err) == ErrSignatureInvalid {
				return nil, ErrSignatureInvalid
			}
			if errors.Cause(err) == ErrSignatureMissing {
				return nil, ErrSignatureMissing
			}
			return nil, errors.Wrap(err, "failed to parse license from file")
		}

		fetchOptions.License = license
	}

	log.ActionWithSpinner("Listing releases")
	v, err := upstream.GetUpdatesUpstream(upstreamURI, &fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to fetch upstream")
	}

	return v, nil
}
