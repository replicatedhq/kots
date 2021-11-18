package pull

import (
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
)

type GetUpdatesOptions struct {
	HelmRepoURI         string
	Namespace           string
	LocalPath           string
	License             *kotsv1beta1.License
	InstalledAt         time.Time
	CurrentCursor       string
	CurrentChannelID    string
	CurrentChannelName  string
	CurrentVersionLabel string
	ReportingInfo       *reportingtypes.ReportingInfo
	Silent              bool
}

// GetUpdates will retrieve all later versions of the application specified in upstreamURI
// using the options specified in getUpdatesOptions. It returns a list of versions.
func GetUpdates(upstreamURI string, getUpdatesOptions GetUpdatesOptions) ([]upstreamtypes.Update, error) {
	log := logger.NewCLILogger()

	if getUpdatesOptions.Silent {
		log.Silence()
	}

	log.Initialize()

	fetchOptions := upstreamtypes.FetchOptions{}
	fetchOptions.HelmRepoURI = getUpdatesOptions.HelmRepoURI
	fetchOptions.LocalPath = getUpdatesOptions.LocalPath
	fetchOptions.InstalledAt = &getUpdatesOptions.InstalledAt
	fetchOptions.CurrentCursor = getUpdatesOptions.CurrentCursor
	fetchOptions.CurrentChannelID = getUpdatesOptions.CurrentChannelID
	fetchOptions.CurrentChannelName = getUpdatesOptions.CurrentChannelName
	fetchOptions.CurrentVersionLabel = getUpdatesOptions.CurrentVersionLabel
	fetchOptions.ReportingInfo = getUpdatesOptions.ReportingInfo

	if getUpdatesOptions.License != nil {
		fetchOptions.License = getUpdatesOptions.License
	}

	log.ActionWithSpinner("Listing releases")
	v, err := upstream.GetUpdatesUpstream(upstreamURI, &fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to peek upstream")
	}

	return v, nil
}
