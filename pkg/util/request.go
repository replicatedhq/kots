package util

import (
	"fmt"
	"io"
	"net/http"

	"github.com/replicatedhq/kots/pkg/buildversion"
)

// NewRequest returns a http.Request object with kots defaults set, including a User-Agent header.
func NewRequest(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to call newrequest: %w", err)
	}

	req.Header.Add("User-Agent", buildversion.GetUserAgent())
	return req, nil
}
