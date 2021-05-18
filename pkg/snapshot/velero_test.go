package snapshot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_imageParse(t *testing.T) {
	req := require.New(t)
	m1 := dockerImageNameRegex.FindStringSubmatch("ttl.sh:443/someuser-eks-airgapped/velero/velero:v1.6.0")
	req.Equal(5, len(m1))
	m2 := dockerImageNameRegex.FindStringSubmatch("velero/velero:v1.6.0")
	req.Equal(5, len(m2))
}
