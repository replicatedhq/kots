package cli

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		cliVersion    string
		apiVersion    string
		expectWarning bool
	}{
		{
			cliVersion:    "alpha",
			apiVersion:    "alpha",
			expectWarning: false,
		},
		{
			cliVersion:    "1.65.0",
			apiVersion:    "test",
			expectWarning: false,
		},
		{
			cliVersion:    "alpha",
			apiVersion:    "1.61.0",
			expectWarning: false,
		},
		{
			cliVersion:    "1.65.0",
			apiVersion:    "v1.61.0",
			expectWarning: true,
		},
		{
			cliVersion:    "v1.55.0",
			apiVersion:    "1.61.0",
			expectWarning: true,
		},
		{
			cliVersion:    "v1.61.0",
			apiVersion:    "1.61.0",
			expectWarning: false,
		},
		{
			cliVersion:    "2022.2.26-nightly-1-g1c2150ef2-dirty",
			apiVersion:    "1.65.0",
			expectWarning: true,
		},
	}

	for _, test := range tests {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		_ = CompareVersions(test.cliVersion, test.apiVersion)

		w.Close()
		output, _ := ioutil.ReadAll(r)
		os.Stderr = oldStderr

		if string(output) != "" && !test.expectWarning {
			t.Errorf("did not expect version mismatch warning for cli version %s and api version %s\n", test.cliVersion, test.apiVersion)
		} else if string(output) == "" && test.expectWarning {
			t.Errorf("expected version mismatch warning for cli version %s and api version %s\n", test.cliVersion, test.apiVersion)
		}
	}
}
