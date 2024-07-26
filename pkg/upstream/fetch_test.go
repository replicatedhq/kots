package upstream

import (
	"encoding/base64"
	"io/ioutil"
	"path/filepath"
	"testing"

	types "github.com/replicatedhq/kots/pkg/upstream/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_downloadUpstream(t *testing.T) {
	req := require.New(t)

	srcDir := t.TempDir()
	workDir := t.TempDir()

	releaseFiles := map[string]string{
		// empty tar.gz file, base64 encoded
		"app.tar.gz": `H4sIAG5/6V0AA+3PSwrCMBQF0CwlK9AkNWY9glQKouBv/RZbnOkss3Mmd3J53LfZhu5SSq3W+Mn9kqns
llzFPNRWSmt1qDHlUnIOsfafFsLz/jjc5inH6XU9n6bLr95cG8c/d9Y/vgkAAAAAAAAAAAAdvQGTLU7p
ACgAAA==`,
	}

	for name, data := range releaseFiles {
		content, err := base64.StdEncoding.DecodeString(data)
		req.NoError(err)
		err = ioutil.WriteFile(filepath.Join(srcDir, name), content, 0644)
		req.NoError(err)
	}

	tests := []struct {
		airgapVersionLabel  string
		currentVersionLabel string
		expectedLabel       string
		expectedChannelID   string
		expectedChannelName string
	}{
		{
			airgapVersionLabel:  "10.9.8",
			currentVersionLabel: "1.2.0",
			expectedLabel:       "10.9.8",
			expectedChannelID:   "channel-2",
			expectedChannelName: "ChannelTwo",
		},
		{
			airgapVersionLabel:  "",
			currentVersionLabel: "1.2.0",
			expectedLabel:       "1.2.0",
			expectedChannelID:   "channel-2",
			expectedChannelName: "ChannelTwo",
		},
	}

	for _, test := range tests {
		fetchOptions := &types.FetchOptions{
			RootDir:             workDir,
			LocalPath:           srcDir,
			CurrentVersionLabel: test.currentVersionLabel,
			License: &kotsv1beta1.License{
				Spec: kotsv1beta1.LicenseSpec{
					Endpoint: "http://localhost",
					AppSlug:  "app-slug",
					Channels: []kotsv1beta1.Channel{
						{
							ChannelID:   "channel-1",
							ChannelName: "ChannelOne",
							ChannelSlug: "channel-one",
							IsDefault:   true,
						},
						{
							ChannelID:   "channel-2",
							ChannelName: "ChannelTwo",
							ChannelSlug: "channel-two",
						},
					},
				},
			},
			Airgap: &kotsv1beta1.Airgap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kots.io/v1beta1",
					Kind:       "Airgap",
				},
				Spec: kotsv1beta1.AirgapSpec{
					AirgapReleaseMeta: kotsv1beta1.AirgapReleaseMeta{
						VersionLabel: test.airgapVersionLabel,
					},
				},
			},
			AppChannelID: "channel-2",
		}
		u, err := FetchUpstream("replicated://app-slug", fetchOptions)
		req.NoError(err)
		assert.Equal(t, test.expectedLabel, u.VersionLabel)
		assert.Equal(t, test.expectedChannelID, u.ChannelID)
		assert.Equal(t, test.expectedChannelName, u.ChannelName)
	}
}
