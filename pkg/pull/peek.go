package pull

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upstream"
)

type PeekOptions struct {
	HelmRepoURI   string
	Namespace     string
	LocalPath     string
	LicenseFile   string
	CurrentCursor string
	Silent        bool
}

// Peek will retrieve all later versions of the application specified in upstreamURI
// using the options specified in peekOptions. It returns a list of versions.
func Peek(upstreamURI string, peekOptions PeekOptions) ([]upstream.Update, error) {
	log := logger.NewLogger()

	if peekOptions.Silent {
		log.Silence()
	}

	log.Initialize()

	fetchOptions := upstream.FetchOptions{}
	fetchOptions.HelmRepoURI = peekOptions.HelmRepoURI
	fetchOptions.LocalPath = peekOptions.LocalPath
	fetchOptions.CurrentCursor = peekOptions.CurrentCursor

	if peekOptions.LicenseFile != "" {
		license, err := parseLicenseFromFile(peekOptions.LicenseFile)
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
	v, err := upstream.PeekUpstream(upstreamURI, &fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to fetch upstream")
	}

	return v, nil
}
