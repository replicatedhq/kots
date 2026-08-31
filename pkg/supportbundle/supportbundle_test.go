package supportbundle

import (
	"fmt"
	"strings"
	"testing"

	"github.com/replicatedhq/kots/pkg/redact"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBundleCommand(t *testing.T) {
	origPodNamespace := util.PodNamespace
	util.PodNamespace = "test-namespace"
	defer func() {
		util.PodNamespace = origPodNamespace
	}()

	appSlug := "test-app"
	command := GetBundleCommand(appSlug)

	require.Len(t, command, 2)
	assert.Equal(t, "curl https://krew.sh/support-bundle | bash", command[0])

	expectedRedactors := strings.Join([]string{
		redact.GetKotsadmRedactSpecURI(),
		redact.GetAppRedactSpecURI(appSlug),
		redact.GetDefaultRedactSpecURI(),
	}, ",")
	assert.Equal(t, fmt.Sprintf("kubectl support-bundle --load-cluster-specs --redactors=%s\n", expectedRedactors), command[1])
}
