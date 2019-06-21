package pullrequest

import (
	"encoding/base64"
	"testing"

	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestNewPullRequestRequest(t *testing.T) {
	tests := []struct {
		describe   string
		watch      *types.Watch
		watchState state.State
		title      string

		owner          string
		repo           string
		branch         string
		path           string
		installationID int

		expectTitle         string
		expectMessage       string
		expectCommitMessage string
	}{
		{
			describe: "version upgrade to 0.2 with release notes",
			watch: &types.Watch{
				Title: "ice-cream",
			},
			watchState: state.State{
				V1: &state.V1{
					Metadata: &state.Metadata{
						Version:      "0.2",
						ReleaseNotes: "rocky mountain",
					},
				},
			},
			title:          "",
			owner:          "o",
			repo:           "r",
			branch:         "b",
			path:           "/",
			installationID: 111,

			expectTitle: "Update ice-cream to version 0.2 from Replicated Ship Cloud",
			expectMessage: `Release notes:

rocky mountain`,
			expectCommitMessage: "ice-cream - 0.2",
		},
		{
			describe: "version upgrade to 0.4 with custom title",
			watch: &types.Watch{
				Title: "hot-dog",
			},
			watchState: state.State{
				V1: &state.V1{
					Metadata: &state.Metadata{
						Version: "0.4",
					},
				},
			},
			title:          "i can put anything here",
			owner:          "o",
			repo:           "r",
			branch:         "b",
			path:           "/",
			installationID: 111,

			expectTitle:         "i can put anything here",
			expectMessage:       "i can put anything here",
			expectCommitMessage: "hot-dog - 0.4",
		},
	}

	req := require.New(t)

	fs := afero.NewMemMapFs()
	mockFile, _ := fs.Create("test-file")
	encoded := `
H4sIAErTtFwAA+3PsQ6CMBSFYWaeoi8AttCWxAdxb6QkJIKmrQNvL9rEwQEnYkz+
bzm36R3OrQ/Bz70Pvq8XN12KPUgprdbimZ01r5RNfmeNEarVxiij9DpL1XbGFkLu
0ubDPSYX1iqTC+etvXVtGDb+8yXinX+iqqrS3caTD3G8zkeRfEzlr0sBAAAAAAAA
AAAAAAAAAL56AF2nh0wAKAAA
`
	fileData, err := base64.StdEncoding.DecodeString(encoded)
	req.NoError(err)
	_, err = mockFile.Write(fileData)
	req.NoError(err)
	mockFile.Sync()

	for _, test := range tests {
		t.Run(test.describe, func(t *testing.T) {
			mockFile.Seek(0, 0) // we are reusing this file...

			prRequest, err := NewPullRequestRequest(test.watch, mockFile, test.owner, test.repo, test.branch, test.path, test.installationID, test.watchState, test.title, "")
			req.NoError(err)

			req.Equal(prRequest.title, test.expectTitle)
			req.Equal(prRequest.message, test.expectMessage)
			req.Equal(prRequest.commitMessage, test.expectCommitMessage)
		})
	}
}
