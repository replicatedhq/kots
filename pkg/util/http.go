package util

import (
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/replicatedhq/kots/pkg/buildversion"
)

var DefaultHTTPClient *retryablehttp.Client

func init() {
	DefaultHTTPClient = retryablehttp.NewClient()
	DefaultHTTPClient.ErrorHandler = errorHandler
}

// NewRequest returns a http.Request object with kots defaults set, including a User-Agent header.
func NewRequest(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to call newrequest: %w", err)
	}

	injectUserAgentHeader(req.Header)
	return req, nil
}

// NewRetryableRequest returns a retryablehttp.Request object with kots defaults set, including a User-Agent header.
func NewRetryableRequest(method string, url string, body io.Reader) (*retryablehttp.Request, error) {
	req, err := retryablehttp.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to call newrequest: %w", err)
	}

	injectUserAgentHeader(req.Header)
	return req, nil
}

func injectUserAgentHeader(header http.Header) {
	header.Add("User-Agent", buildversion.GetUserAgent())
}

// errorHandler mimics net/http rather than doing anything fancy like the retryablehttp library.
func errorHandler(resp *http.Response, err error, attempt int) (*http.Response, error) {
	return resp, err
}
