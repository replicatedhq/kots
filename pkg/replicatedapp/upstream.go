package replicatedapp

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
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

func (r *ReplicatedUpstream) GetRequest(method string, license *kotsv1beta1.License, cursor string, selectedChannelID string) (*http.Request, error) {
	endpoint, err := getReplicatedAppEndpoint(license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get replicated app endpoint")
	}

	urlPath := path.Join("release", license.Spec.AppSlug)
	if r.Channel != nil {
		urlPath = path.Join(urlPath, *r.Channel)
	}

	urlValues := url.Values{}
	urlValues.Set("channelSequence", cursor)
	if r.VersionLabel != nil {
		urlValues.Set("versionLabel", *r.VersionLabel)
	}
	urlValues.Add("licenseSequence", fmt.Sprintf("%d", license.Spec.LicenseSequence))
	urlValues.Add("isSemverSupported", "true")
	urlValues.Add("selectedChannelId", selectedChannelID)

	url := fmt.Sprintf("%s/%s?%s", endpoint, urlPath, urlValues.Encode())

	req, err := util.NewRequest(method, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID)))))

	return req, nil
}

func getReplicatedAppEndpoint(license *kotsv1beta1.License) (string, error) {
	var endpoint string

	if util.IsEmbeddedCluster() {
		endpoint = os.Getenv("REPLICATED_API_ENDPOINT")
		if endpoint == "" {
			return "", errors.New("REPLICATED_API_ENDPOINT environment variable is required")
		}
	} else {
		endpoint = license.Spec.Endpoint
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = fmt.Sprintf("https://%s", endpoint)
		}
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.Wrap(err, "parse endpoint")
	}

	host := u.Hostname()
	if u.Port() != "" {
		host = net.JoinHostPort(u.Hostname(), u.Port())
	}

	return fmt.Sprintf("%s://%s", u.Scheme, host), nil
}
