package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.podman.io/image/v5/types"
)

func TestSetSystemRegistriesConfPath_UsesHostV2Config(t *testing.T) {
	resetRegistriesConfPathOnce()

	dir := t.TempDir()
	hostPath := filepath.Join(dir, "registries.conf")
	// A minimal valid v2 registries.conf.
	content := `unqualified-search-registries = ["registry.access.redhat.com"]
`
	require.NoError(t, os.WriteFile(hostPath, []byte(content), 0644))

	t.Setenv("CONTAINERS_REGISTRIES_CONF", hostPath)

	sys := &types.SystemContext{}
	require.NoError(t, SetSystemRegistriesConfPath(sys))
	assert.Equal(t, hostPath, sys.SystemRegistriesConfPath)
}

func TestSetSystemRegistriesConfPath_FallsBackWhenHostConfigIsV1(t *testing.T) {
	resetRegistriesConfPathOnce()

	dir := t.TempDir()
	hostPath := filepath.Join(dir, "registries.conf")
	// A v1 registries.conf (uses the v1-only [registries.search] block).
	content := `[registries.search]
registries = ['registry.access.redhat.com']
`
	require.NoError(t, os.WriteFile(hostPath, []byte(content), 0644))

	t.Setenv("CONTAINERS_REGISTRIES_CONF", hostPath)

	sys := &types.SystemContext{}
	require.NoError(t, SetSystemRegistriesConfPath(sys))
	assert.NotEqual(t, hostPath, sys.SystemRegistriesConfPath)
	assert.NotEmpty(t, sys.SystemRegistriesConfPath)
}

func TestSetSystemRegistriesConfPath_FallsBackWhenHostConfigMissing(t *testing.T) {
	resetRegistriesConfPathOnce()

	hostPath := filepath.Join(t.TempDir(), "does-not-exist.conf")
	t.Setenv("CONTAINERS_REGISTRIES_CONF", hostPath)

	sys := &types.SystemContext{}
	require.NoError(t, SetSystemRegistriesConfPath(sys))
	assert.NotEmpty(t, sys.SystemRegistriesConfPath)
}
