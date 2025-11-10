package replicatedapp

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
)

type ReplicatedUpstream struct {
	Channel      *string
	AppSlug      string
	VersionLabel *string
}

func ParseReplicatedURL(u *url.URL) (*ReplicatedUpstream, error) {
	replicatedUpstream := ReplicatedUpstream{}

	if u.User != nil {
		if u.User.Username() != "" {
			replicatedUpstream.AppSlug = u.User.Username()
			versionLabel := u.Hostname()
			replicatedUpstream.VersionLabel = &versionLabel
		}
	}

	if replicatedUpstream.AppSlug == "" {
		replicatedUpstream.AppSlug = u.Hostname()
		if u.Path != "" {
			channel := strings.TrimPrefix(u.Path, "/")
			replicatedUpstream.Channel = &channel
		}
	}

	return &replicatedUpstream, nil
}

func (r *ReplicatedUpstream) GetRequest(method string, license *licensewrapper.LicenseWrapper, cursor string, selectedChannelID string) (*retryablehttp.Request, error) {
	endpoint, err := getReplicatedAppEndpoint(license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get replicated app endpoint")
	}

	urlPath := path.Join("release", license.GetAppSlug())
	if r.Channel != nil {
		urlPath = path.Join(urlPath, *r.Channel)
	}

	urlValues := url.Values{}
	urlValues.Set("channelSequence", cursor)
	if r.VersionLabel != nil {
		urlValues.Set("versionLabel", *r.VersionLabel)
	}
	urlValues.Add("licenseSequence", fmt.Sprintf("%d", license.GetLicenseSequence()))
	urlValues.Add("isSemverSupported", "true")
	urlValues.Add("isEmbeddedCluster", fmt.Sprintf("%t", util.IsEmbeddedCluster()))
	urlValues.Add("selectedChannelId", selectedChannelID)

	url := fmt.Sprintf("%s/%s?%s", endpoint, urlPath, urlValues.Encode())

	req, err := util.NewRetryableRequest(method, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	licenseID := license.GetLicenseID()
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", licenseID, licenseID)))))

	return req, nil
}

func getReplicatedAppEndpoint(license *licensewrapper.LicenseWrapper) (string, error) {
	endpoint := util.ReplicatedAppEndpoint(license)

	u, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse endpoint")
	}

	host := u.Hostname()
	if u.Port() != "" {
		host = net.JoinHostPort(u.Hostname(), u.Port())
	}

	return fmt.Sprintf("%s://%s", u.Scheme, host), nil
}
