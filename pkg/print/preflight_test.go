package print

import (
	"os"
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	"github.com/stretchr/testify/require"
)

func TestPreflightNoticeIsPrintedWithoutCheckResults(t *testing.T) {
	output, err := os.CreateTemp(t.TempDir(), "output")
	require.NoError(t, err)
	original := os.Stdout
	os.Stdout = output
	defer func() { os.Stdout = original; output.Close() }()

	PreflightResults(preflighttypes.PreflightResults{Notice: kotsutil.UnsupportedTroubleshootMessage})
	printed, err := os.ReadFile(output.Name())
	require.NoError(t, err)
	require.Contains(t, string(printed), kotsutil.UnsupportedTroubleshootMessage)
}
