package upstream

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseHelmURL(t *testing.T) {
	tests := []struct {
		name                 string
		uri                  string
		expectedRepo         string
		expectedChartName    string
		expectedChartVersion string
	}{
		{
			name:                 "stable/mysql",
			uri:                  "helm://stable/mysql",
			expectedRepo:         "stable",
			expectedChartName:    "mysql",
			expectedChartVersion: "",
		},
		{
			name:                 "stable/mysql@1.3.1",
			uri:                  "helm://stable/mysql@1.3.1",
			expectedRepo:         "stable",
			expectedChartName:    "mysql",
			expectedChartVersion: "1.3.1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			u, err := url.ParseRequestURI(test.uri)
			req.NoError(err)

			repo, name, version, err := parseHelmURL(u)
			req.NoError(err)
			assert.Equal(t, test.expectedRepo, repo)
			assert.Equal(t, test.expectedChartName, name)
			assert.Equal(t, test.expectedChartVersion, version)
		})
	}
}
