package util

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
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

// errorHandler includes the body in the response when there is an error
func errorHandler(resp *http.Response, err error, attempt int) (*http.Response, error) {
	var req http.Request
	var bodyStr string

	if resp != nil && resp.Request != nil {
		req = *resp.Request

		body, readErr := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if readErr != nil {
			err = errors.Wrap(err, fmt.Sprintf("failed to read response body: %v", readErr))
		} else {
			bodyStr = strings.TrimSpace(string(body))
		}
	}

	// this means CheckRetry thought the request was a failure, but didn't communicate why
	if err == nil {
		if bodyStr != "" {
			return resp, fmt.Errorf("%s %s giving up after %d attempt(s): %s",
				req.Method, redactURL(req.URL), attempt, bodyStr)
		}
		return resp, fmt.Errorf("%s %s giving up after %d attempt(s)",
			req.Method, redactURL(req.URL), attempt)
	}

	if bodyStr != "" {
		return resp, fmt.Errorf("%s %s giving up after %d attempt(s) with error %w: %s",
			req.Method, redactURL(req.URL), attempt, err, bodyStr)
	}
	return resp, fmt.Errorf("%s %s giving up after %d attempt(s) with error %w",
		req.Method, redactURL(req.URL), attempt, err)
}

// Taken from url.URL#Redacted() which was introduced in go 1.15.
// We can switch to using it directly if we'll bump the minimum required go version.
func redactURL(u *url.URL) string {
	if u == nil {
		return ""
	}

	ru := *u
	if _, has := ru.User.Password(); has {
		ru.User = url.UserPassword(ru.User.Username(), "xxxxx")
	}
	return ru.String()
}
