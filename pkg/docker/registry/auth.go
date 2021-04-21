package registry

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/containers/image/v5/pkg/docker/config"
	"github.com/containers/image/v5/types"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/logger"
)

type ScopeAction string

var (
	insecureClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Proxy: http.ProxyFromEnvironment,
		},
	}
)

const (
	ActionPull ScopeAction = "pull"
	ActionPush ScopeAction = "push"
)

func LoadAuthForRegistry(endpoint string) (string, string, error) {
	sys := &types.SystemContext{DockerDisableV1Ping: true}
	username, password, err := config.GetAuthentication(sys, endpoint)
	if err != nil {
		return "", "", errors.Wrapf(err, "error loading username and password")
	}

	return username, password, nil
}

func CheckAccess(endpoint, username, password, org string, requestedAction ScopeAction) error {

	endpoint = sanitizeEndpoint(endpoint)

	// We need to check if we can push images to a repo.
	// We cannot get push permission to an org alone.
	scope := org + "/testrepo"
	basicAuthToken := makeBasicAuthToken(username, password)

	if IsECREndpoint(endpoint) {
		token, err := GetECRBasicAuthToken(endpoint, username, password)
		if err != nil {
			return errors.Wrap(err, "failed to ping registry")
		}
		basicAuthToken = token
		scope = org // ECR has no concept of organization and it should be an empty string
	}

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

	host := challenges[0].Parameters["realm"]
	v := url.Values{}
	v.Set("service", challenges[0].Parameters["service"])
	v.Set("scope", fmt.Sprintf("repository:%s:%s", scope, requestedAction))

	authURL := host + "?" + v.Encode()

	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create auth request")
	}

	req.Header.Add("User-Agent", fmt.Sprintf("KOTS/%s", buildversion.Version()))
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

	bearerToken, err := newBearerTokenFromJSONBlob(authBody)
	if err != nil {
		// not a fatal error - some registries don't return JWTs
		logger.NewCLILogger().Info("failed to parse registry auth bearer token, continuing anyways: %s", err.Error())
		return nil
	}

	jwtToken, err := bearerToken.getJwtToken()
	if err != nil {
		// not a fatal error - some registries don't return JWTs
		logger.NewCLILogger().Info("failed to parse registry auth jwt, continuing anyways: %s", err.Error())
		return nil
	}

	claims, err := getJwtTokenClaims(jwtToken)
	if err != nil {
		// not a fatal error - some registries don't return JWTs
		logger.NewCLILogger().Info("failed to find registry auth claims in jwt, continuing anyways: %s", err.Error())
		return nil
	}

	// If requested access is "push", we need to check that we actually got it
	for _, access := range claims.Access {
		if access.Type != "repository" {
			continue
		}
		if access.Name != scope {
			continue
		}
		for _, action := range access.Actions {
			if action == string(requestedAction) {
				return nil
			}
		}
	}

	return errors.Errorf("%q has no %s permission in %q", username, requestedAction, org)
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
