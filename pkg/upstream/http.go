package upstream

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type PrivateUpstream struct {
	Endpoint string
	AppSlug  string
}

type PrivateRelease struct {
	Sequence     int       `json:"sequence"`
	VersionLabel string    `json:"versionLabel"`
	CreatedAt    time.Time `json:"createdAt"`
}

func getUpdatesHttp(u *url.URL, localPath string, currentCursor ReplicatedCursor, versionLabel string, unsignedLicense *kotsv1beta1.UnsignedLicense) ([]Update, error) {
	if localPath != "" {
		parsedLocalRelease, err := readReplicatedAppFromLocalPath(localPath, currentCursor, versionLabel)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read private app from local path")
		}

		return []Update{{Cursor: parsedLocalRelease.UpdateCursor.Cursor, VersionLabel: versionLabel}}, nil
	}

	// We currently require a private license
	if unsignedLicense == nil {
		return nil, errors.New("UnsignedLicense is required")
	}

	privateUpstream, err := parseHTTPURL(u.RequestURI())
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse http upstream")
	}

	pendingReleases, err := listPendingPrivateAppReleases(privateUpstream, unsignedLicense, currentCursor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list pending private app releases")
	}

	updates := []Update{}
	for _, pendingRelease := range pendingReleases {
		updates = append(updates, Update{
			Cursor:       strconv.Itoa(pendingRelease.Sequence),
			VersionLabel: pendingRelease.VersionLabel,
		})
	}

	return updates, nil
}

func listPendingPrivateAppReleases(privateUpstream *PrivateUpstream, unsignedLicense *kotsv1beta1.UnsignedLicense, cursor ReplicatedCursor) ([]PrivateRelease, error) {
	u, err := url.Parse(unsignedLicense.Spec.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse endpoint from private license")
	}

	hostname := u.Hostname()
	if u.Port() != "" {
		hostname = fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	}

	sequence := cursor.Cursor

	urlValues := url.Values{}
	urlValues.Set("sequence", sequence)
	url := fmt.Sprintf("%s://%s/release/%s/pending?%s", u.Scheme, hostname, unsignedLicense.Spec.Slug, urlValues.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	req.Header.Add("User-Agent", fmt.Sprintf("KOTS/%s", version.Version()))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode >= 400 {
		if len(body) > 0 {
			return nil, util.ActionableError{Message: string(body)}
		}
		return nil, errors.Errorf("unexpected result from get request: %d", resp.StatusCode)
	}

	privateReleases := []PrivateRelease{}
	if err := json.Unmarshal(body, &privateReleases); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	return privateReleases, nil
}

func downloadHttp(httpURI string, fetchOptions *FetchOptions, existingConfigValues *kotsv1beta1.ConfigValues, updateCursor ReplicatedCursor, cipher *crypto.AESCipher) (*types.Upstream, error) {
	// a license file is required
	if fetchOptions.UnsignedLicense == nil {
		return nil, errors.New("A private license file is required")
	}

	privateUpstream, err := parseHTTPURL(httpURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parsed http upstream")
	}

	release, err := downloadHTTPApp(privateUpstream, fetchOptions.UnsignedLicense, updateCursor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download http app")
	}

	// find the config in the upstream and write out the default values
	application := findAppInRelease(release)

	config, _, _, _, err := findTemplateContextDataInRelease(release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find config in release")
	}
	if config != nil || existingConfigValues != nil {
		// If config existed and was removed from the app,
		// values will be carried over to the new version anyway.
		configValues, err := createConfigValues(application.Name, config, existingConfigValues, cipher, nil, fetchOptions.UnsignedLicense)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create empty config values")
		}

		release.Manifests["userdata/config.yaml"] = mustMarshalConfigValues(configValues)
	}

	release.Manifests["userdata/license.yaml"] = MustMarshalUnsignedLicense(fetchOptions.UnsignedLicense)

	files, err := releaseToFiles(release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get files from release")
	}

	upstream := &types.Upstream{
		URI:           httpURI,
		Name:          application.Name,
		Files:         files,
		Type:          "private",
		UpdateCursor:  release.UpdateCursor.Cursor,
		EncryptionKey: cipher.ToString(),
	}

	return upstream, nil

}

func MustMarshalUnsignedLicense(unsignedLicense *kotsv1beta1.UnsignedLicense) []byte {
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(unsignedLicense, &b); err != nil {
		panic(err)
	}

	return b.Bytes()
}

func parseHTTPURL(url string) (*PrivateUpstream, error) {
	parts := strings.Split(url, "/")

	privateUpstream := PrivateUpstream{
		Endpoint: url,
		AppSlug:  parts[len(parts)-1],
	}

	return &privateUpstream, nil
}

func downloadHTTPApp(privateUpstream *PrivateUpstream, unsignedLicense *kotsv1beta1.UnsignedLicense, cursor ReplicatedCursor) (*Release, error) {
	getReq, err := privateUpstream.getRequest("GET", unsignedLicense, cursor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer getResp.Body.Close()

	if getResp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(getResp.Body)
		if len(body) > 0 {
			return nil, util.ActionableError{Message: string(body)}
		}
		return nil, errors.Errorf("unexpected result from get request: %d", getResp.StatusCode)
	}

	updateSequence := getResp.Header.Get("X-KOTS-ChannelSequence")
	updateChannel := getResp.Header.Get("X-KOTS-ChannelName")
	versionLabel := getResp.Header.Get("X-KOTS-VersionLabel")

	gzf, err := gzip.NewReader(getResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new gzip reader")
	}

	release := Release{
		Manifests: make(map[string][]byte),
		UpdateCursor: ReplicatedCursor{
			ChannelName: updateChannel,
			Cursor:      updateSequence,
		},
		VersionLabel: versionLabel,
	}
	tarReader := tar.NewReader(gzf)
	i := 0
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to get next file from reader")
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			content, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read file from tar")
			}

			release.Manifests[name] = content
		}

		i++
	}

	return &release, nil

}

func (p *PrivateUpstream) getRequest(method string, unsignedLicense *kotsv1beta1.UnsignedLicense, cursor ReplicatedCursor) (*http.Request, error) {
	u, err := url.Parse(unsignedLicense.Spec.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse endpoint from license")
	}

	hostname := u.Hostname()
	if u.Port() != "" {
		hostname = fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	}

	urlPath := path.Join(hostname, "release", unsignedLicense.Spec.Slug)

	urlValues := url.Values{}
	urlValues.Set("channelSequence", cursor.Cursor)
	url := fmt.Sprintf("%s://%s?%s", u.Scheme, urlPath, urlValues.Encode())

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	req.Header.Add("User-Agent", fmt.Sprintf("KOTS/%s", version.Version()))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", unsignedLicense.Name, unsignedLicense.Name)))))

	return req, nil
}
