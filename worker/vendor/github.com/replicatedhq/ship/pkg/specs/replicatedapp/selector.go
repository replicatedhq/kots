package replicatedapp

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/google/go-querystring/query"
)

// Selector selects a replicated.app spec from the Vendor's releases and channels.
// See pkg/cli/root.go for some more info on which are required and why.
//
// note that `url` struct tags are only for serialize, they don't work for deserialize
type Selector struct {
	// required
	CustomerID     string `url:"customer_id,omitempty"`
	InstallationID string `url:"installation_id,omitempty"`
	// OR
	AppSlug string `url:"AppSlug,omitempty"`

	// optional
	Upstream      string `url:"upstream,omitempty"`
	ReleaseID     string `url:"release_id,omitempty"` // NOTE: this is unused
	ReleaseSemver string `url:"release_semver,omitempty"`
	LicenseID     string `url:"license_id,omitempty"`
}

func (s *Selector) String() string {
	sCopy := *s
	sCopy.AppSlug = ""

	v, err := query.Values(sCopy)
	if err != nil {
		return "Selector{(failed to parse)}"
	}
	return v.Encode()
}

var pathQuery = regexp.MustCompile(`replicated\.app/([\w_\-/]+[\w_\-]+)`)

// this is less janky
func (s *Selector) UnmarshalFrom(url *url.URL) *Selector {
	for key, values := range url.Query() {
		if len(values) == 0 {
			continue
		}
		switch key {
		case "customer_id":
			s.CustomerID = values[0]
		case "installation_id":
			s.InstallationID = values[0]
		case "release_id":
			s.ReleaseID = values[0]
		case "release_semver":
			s.ReleaseSemver = values[0]
		case "license_id":
			s.LicenseID = values[0]
		}
	}

	if pathQuery.MatchString(url.Path) && s.CustomerID == "" {
		matches := pathQuery.FindStringSubmatch(url.Path)
		if len(matches) == 2 {
			s.AppSlug = matches[1]
		}
	}

	if strings.HasPrefix(url.String(), "staging.replicated.app") {
		s.Upstream = "https://pg.staging.replicated.com/graphql"
	}
	return s
}

func (s *Selector) GetBasicAuthUsername() string {
	if s.CustomerID != "" {
		return s.CustomerID
	}
	return s.LicenseID
}
