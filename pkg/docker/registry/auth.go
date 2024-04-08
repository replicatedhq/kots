package registry

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/containers/image/v5/pkg/docker/config"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry/challenge"
	"github.com/replicatedhq/kots/pkg/util"
)

var (
	insecureClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Proxy: http.ProxyFromEnvironment,
		},
	}
)

func LoadAuthForRegistry(endpoint string) (string, string, error) {
	sys := &types.SystemContext{DockerDisableV1Ping: true}
	username, password, err := config.GetAuthentication(sys, endpoint)
	if err != nil {
		return "", "", errors.Wrapf(err, "error loading username and password")
	}

	return username, password, nil
}

func CheckAccess(endpoint, username, password string) error {
	endpoint = sanitizeEndpoint(endpoint)

	pingURL := fmt.Sprintf("https://%s/v2/", endpoint)
	resp, err := insecureClient.Get(pingURL)
	if err != nil {
		// attempt with http
		pingURL = fmt.Sprintf("http://%s/v2/", endpoint)
		resp, err = insecureClient.Get(pingURL)
		if err != nil {
			return errors.Wrap(err, "failed to ping registry")
		}
	}

	if resp.StatusCode == http.StatusOK {
		// Anonymous registry that does not require authentication
		return nil
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return errors.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	challenges := challenge.ResponseChallenges(resp)
	if len(challenges) == 0 {
		return errors.Wrap(err, "no auth challenges found for endpoint")
	}

	if challenges[0].Scheme == "basic" {
		// ecr uses basic auth. not much more we can do here without actually pushing an image
		return nil
	}

	authURL := challenges[0].Parameters["realm"]
	basicAuthToken := makeBasicAuthToken(username, password)

	// some registries (e.g. ACR - Azure Container Registry) require the "service" parameter to be set.
	if service := challenges[0].Parameters["service"]; service != "" {
		authURL = fmt.Sprintf("%s?service=%s", authURL, service)
	}

	if IsECREndpoint(endpoint) && username != "AWS" {
		token, err := GetECRBasicAuthToken(endpoint, username, password)
		if err != nil {
			return errors.Wrap(err, "failed to ping registry")
		}
		basicAuthToken = token
	}

	req, err := util.NewRequest("GET", authURL, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create auth request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", basicAuthToken))

	resp, err = insecureClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute auth request")
	}
	defer resp.Body.Close()

	authBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to load auth response")
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(errorResponseToString(resp.StatusCode, authBody))
	}

	// We don't validate JWT tokens here because some registries don't return any,
	// and some registries return JWT tokens without access scopes.

	return nil
}

func makeBasicAuthToken(username, password string) string {
	token := fmt.Sprintf("%s:%s", username, password)
	return base64.StdEncoding.EncodeToString([]byte(token))
}

func sanitizeEndpoint(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimSuffix(endpoint, "/v2/")
	endpoint = strings.TrimSuffix(endpoint, "/v2")
	endpoint = strings.TrimSuffix(endpoint, "/v1/")
	endpoint = strings.TrimSuffix(endpoint, "/v1")
	if endpoint == "docker.io" {
		endpoint = "index.docker.io"
	}
	return endpoint
}
