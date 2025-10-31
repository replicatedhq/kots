package pull

import (
	"os"
	"time"

	"github.com/pkg/errors"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
)

type GetUpdatesOptions struct {
	HelmRepoURI        string
	Namespace          string
	License            licensewrapper.LicenseWrapper
	LastUpdateCheckAt  *time.Time
	CurrentCursor      string
	CurrentChannelID   string
	CurrentChannelName string
	ChannelChanged     bool
	ReportingInfo      *reportingtypes.ReportingInfo
	Silent             bool
}

// GetUpdates will retrieve all later versions of the application specified in upstreamURI
// using the options specified in getUpdatesOptions. It returns a list of versions.
func GetUpdates(upstreamURI string, getUpdatesOptions GetUpdatesOptions) (*upstreamtypes.UpdateCheckResult, error) {
	log := logger.NewCLILogger(os.Stdout)

	if getUpdatesOptions.Silent {
		log.Silence()
	}

	log.Initialize()

	fetchOptions := upstreamtypes.FetchOptions{}
	fetchOptions.HelmRepoURI = getUpdatesOptions.HelmRepoURI
	fetchOptions.LastUpdateCheckAt = getUpdatesOptions.LastUpdateCheckAt
	fetchOptions.CurrentCursor = getUpdatesOptions.CurrentCursor
	fetchOptions.CurrentChannelID = getUpdatesOptions.CurrentChannelID
	fetchOptions.CurrentChannelName = getUpdatesOptions.CurrentChannelName
	fetchOptions.ChannelChanged = getUpdatesOptions.ChannelChanged
	fetchOptions.ReportingInfo = getUpdatesOptions.ReportingInfo

	if getUpdatesOptions.License.IsV1() || getUpdatesOptions.License.IsV2() {
		fetchOptions.License = getUpdatesOptions.License
	}

	log.ActionWithSpinner("Listing releases")
	v, err := upstream.GetUpdatesUpstream(upstreamURI, &fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return nil, errors.Wrap(err, "failed to peek upstream")
	}

	log.FinishSpinner()

	return v, nil
}
