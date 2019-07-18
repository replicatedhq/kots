package replicatedapp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/viper"
)

const ShipRelease = `
    id
    sequence
    channelId
    channelName
    channelIcon
    semver
    releaseNotes
    spec
    images {
      url
      source
      appSlug
      imageKey
    }
    githubContents {
      repo
      path
      ref
      files {
        name
        path
        sha
        size
        data
      }
    }
    entitlements {
      values {
        key
        value
        labels {
          key
          value
        }
      }
      utilizations {
        key
        value
      }
      meta {
        lastUpdated
        customerID
        installationID
      }
      serialized
      signature
    }
    created
    registrySecret
    collectSpec
    analyzeSpec`

const GetAppspecQuery = `
query($semver: String) {
  shipRelease (semver: $semver) {
` + ShipRelease + `
  }
}`

const GetSlugAppSpecQuery = `
query($appSlug: String!, $licenseID: String, $releaseID: String, $semver: String) {
  shipSlugRelease (appSlug: $appSlug, licenseID: $licenseID, releaseID: $releaseID, semver: $semver) {
` + ShipRelease + `
  }
}`

const GetLicenseQuery = `
query($licenseId: String) {
  license (licenseId: $licenseId) {
    id
    assignee
    createdAt
    expiresAt
    type
  }
}`

const RegisterInstallQuery = `
mutation($channelId: String!, $releaseId: String!) {
  shipRegisterInstall(
    channelId: $channelId
    releaseId: $releaseId
  )
}`

// GraphQLClient is a client for the graphql Payload API
type GraphQLClient struct {
	GQLServer *url.URL
	Client    *http.Client
}

// GraphQLRequest is a json-serializable request to the graphql server
type GraphQLRequest struct {
	Query         string            `json:"query"`
	Variables     map[string]string `json:"variables"`
	OperationName string            `json:"operationName"`
}

// GraphQLError represents an error returned by the graphql server
type GraphQLError struct {
	Locations []map[string]interface{} `json:"locations"`
	Message   string                   `json:"message"`
	Code      string                   `json:"code"`
}

// GQLLicenseResponse is the top-level response object from the graphql server
type GQLGetLicenseResponse struct {
	Data   LicenseWrapper `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GQLGetReleaseResponse is the top-level response object from the graphql server
type GQLGetReleaseResponse struct {
	Data   ShipReleaseWrapper `json:"data,omitempty"`
	Errors []GraphQLError     `json:"errors,omitempty"`
}

// GQLGetSlugReleaseResponse is the top-level response object from the graphql server
type GQLGetSlugReleaseResponse struct {
	Data   ShipSlugReleaseWrapper `json:"data,omitempty"`
	Errors []GraphQLError         `json:"errors,omitempty"`
}

// ShipReleaseWrapper wraps the release response form GQL
type LicenseWrapper struct {
	License license `json:"license"`
}

// ShipReleaseWrapper wraps the release response form GQL
type ShipReleaseWrapper struct {
	ShipRelease state.ShipRelease `json:"shipRelease"`
}

// ShipSlugReleaseWrapper wraps the release response form GQL
type ShipSlugReleaseWrapper struct {
	ShipSlugRelease state.ShipRelease `json:"shipSlugRelease"`
}

// GQLRegisterInstallResponse is the top-level response object from the graphql server
type GQLRegisterInstallResponse struct {
	Data struct {
		ShipRegisterInstall bool `json:"shipRegisterInstall"`
	} `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

func parseServerTS(ts string) time.Time {
	parsed, _ := time.Parse("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)", ts)
	return parsed
}

type license struct {
	ID        string `json:"id"`
	Assignee  string `json:"assignee"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
	Type      string `json:"type"`
}

func (l *license) ToLicenseMeta() api.License {
	return api.License{
		ID:        l.ID,
		Assignee:  l.Assignee,
		CreatedAt: parseServerTS(l.CreatedAt),
		ExpiresAt: parseServerTS(l.ExpiresAt),
		Type:      l.Type,
	}
}

func (l *license) ToStateLicense() *state.License {
	return &state.License{
		ID:        l.ID,
		Assignee:  l.Assignee,
		CreatedAt: parseServerTS(l.CreatedAt),
		ExpiresAt: parseServerTS(l.ExpiresAt),
		Type:      l.Type,
	}
}

type callInfo struct {
	username string
	password string
	request  GraphQLRequest
	upstream string
}

// NewGraphqlClient builds a new client using a viper instance
func NewGraphqlClient(v *viper.Viper, client *http.Client) (*GraphQLClient, error) {
	addr := v.GetString("customer-endpoint")
	server, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse GQL server address %s", addr)
	}
	return &GraphQLClient{
		GQLServer: server,
		Client:    client,
	}, nil
}

// GetRelease gets a payload from the graphql server
func (c *GraphQLClient) GetRelease(selector *Selector) (*state.ShipRelease, error) {
	requestObj := GraphQLRequest{
		Query: GetAppspecQuery,
		Variables: map[string]string{
			"semver": selector.ReleaseSemver,
		},
	}

	ci := callInfo{
		username: selector.GetBasicAuthUsername(),
		password: selector.InstallationID,
		request:  requestObj,
		upstream: selector.Upstream,
	}

	shipResponse := &GQLGetReleaseResponse{}
	if err := c.callGQL(ci, shipResponse); err != nil {
		return nil, err
	}

	if shipResponse.Errors != nil && len(shipResponse.Errors) > 0 {
		var multiErr *multierror.Error
		for _, err := range shipResponse.Errors {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s: %s", err.Code, err.Message))

		}
		return nil, multiErr.ErrorOrNil()
	}

	return &shipResponse.Data.ShipRelease, nil
}

// GetSlugRelease gets a release from the graphql server by app slug
func (c *GraphQLClient) GetSlugRelease(selector *Selector) (*state.ShipRelease, error) {
	requestObj := GraphQLRequest{
		Query: GetSlugAppSpecQuery,
		Variables: map[string]string{
			"appSlug":   selector.AppSlug,
			"licenseID": selector.LicenseID,
			"releaseID": selector.ReleaseID,
			"semver":    selector.ReleaseSemver,
		},
	}

	ci := callInfo{
		username: selector.GetBasicAuthUsername(),
		password: selector.InstallationID,
		request:  requestObj,
		upstream: selector.Upstream,
	}

	shipResponse := &GQLGetSlugReleaseResponse{}
	if err := c.callGQL(ci, shipResponse); err != nil {
		return nil, err
	}

	if shipResponse.Errors != nil && len(shipResponse.Errors) > 0 {
		var multiErr *multierror.Error
		for _, err := range shipResponse.Errors {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s: %s", err.Code, err.Message))

		}
		return nil, multiErr.ErrorOrNil()
	}

	return &shipResponse.Data.ShipSlugRelease, nil
}

func (c *GraphQLClient) GetLicense(selector *Selector) (*license, error) {
	requestObj := GraphQLRequest{
		Query: GetLicenseQuery,
		Variables: map[string]string{
			"licenseId": selector.LicenseID,
		},
	}

	ci := callInfo{
		username: selector.GetBasicAuthUsername(),
		password: selector.InstallationID,
		request:  requestObj,
		upstream: selector.Upstream,
	}

	licenseResponse := &GQLGetLicenseResponse{}
	if err := c.callGQL(ci, licenseResponse); err != nil {
		return nil, err
	}

	if len(licenseResponse.Errors) > 0 {
		var multiErr *multierror.Error
		for _, err := range licenseResponse.Errors {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s: %s", err.Code, err.Message))

		}
		return nil, multiErr.ErrorOrNil()
	}

	return &licenseResponse.Data.License, nil
}

func (c *GraphQLClient) RegisterInstall(customerID, installationID, channelID, releaseID string) error {
	requestObj := GraphQLRequest{
		Query: RegisterInstallQuery,
		Variables: map[string]string{
			"channelId": channelID,
			"releaseId": releaseID,
		},
	}

	ci := callInfo{
		username: customerID,
		password: installationID,
		request:  requestObj,
	}

	shipResponse := &GQLRegisterInstallResponse{}
	if err := c.callGQL(ci, shipResponse); err != nil {
		return err
	}

	if shipResponse.Errors != nil && len(shipResponse.Errors) > 0 {
		var multiErr *multierror.Error
		for _, err := range shipResponse.Errors {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s: %s", err.Code, err.Message))

		}
		return multiErr.ErrorOrNil()
	}

	return nil
}

func (c *GraphQLClient) callGQL(ci callInfo, result interface{}) error {
	body, err := json.Marshal(ci.request)
	if err != nil {
		return errors.Wrap(err, "marshal request")
	}

	bodyReader := ioutil.NopCloser(bytes.NewReader(body))

	gqlServer := c.GQLServer.String()
	if ci.upstream != "" {
		gqlServer = ci.upstream
	}

	graphQLRequest, err := http.NewRequest(http.MethodPost, gqlServer, bodyReader)
	if err != nil {
		return errors.Wrap(err, "create new request")
	}

	graphQLRequest.Header = map[string][]string{
		"Content-Type": {"application/json"},
	}

	if ci.username != "" || ci.password != "" {
		authString := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", ci.username, ci.password)))
		graphQLRequest.Header["Authorization"] = []string{"Basic " + authString}
	}

	resp, err := c.Client.Do(graphQLRequest)
	if err != nil {
		return errors.Wrap(err, "send request")
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "read body")
	}

	if err := json.Unmarshal(responseBody, result); err != nil {
		return errors.Wrapf(err, "unmarshal response %s", responseBody)
	}

	return nil
}
