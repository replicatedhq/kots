package reporting

import (
	"encoding/json"
	"testing"

	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	"github.com/stretchr/testify/require"
)

func TestPreflightReportingRetainsNotices(t *testing.T) {
	for _, tc := range []struct{ result, state string }{
		{`null`, ""},
		{`{}`, "pass"},
		{`{"notice":"Skipped checks"}`, "warn"},
		{`{"notice":"Skipped checks","results":[{"isPass":true}]}`, "warn"},
		{`{"results":[{"isPass":true}]}`, "pass"},
		{`{"results":[{"isWarn":true}]}`, "warn"},
		{`{"notice":"Skipped checks","results":[{"isFail":true}]}`, "fail"},
		{`{"notice":"Skipped checks","errors":[{"error":"collection failed"}]}`, "fail"},
	} {
		t.Run(tc.result, func(t *testing.T) {
			var result *preflighttypes.PreflightResults
			require.NoError(t, json.Unmarshal([]byte(tc.result), &result))
			require.Equal(t, tc.state, getPreflightState(result))
		})
	}
}
