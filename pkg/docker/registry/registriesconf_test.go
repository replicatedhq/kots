package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.podman.io/image/v5/pkg/sysregistriesv2"
	"go.podman.io/image/v5/types"
)

func TestSetSystemRegistriesConfPath_UsesHostV2Config(t *testing.T) {
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

	// Verify the generated v2 file preserves the search registry.
	ctx := &types.SystemContext{SystemRegistriesConfPath: sys.SystemRegistriesConfPath}
	v2, err := sysregistriesv2.TryUpdatingCache(ctx)
	require.NoError(t, err)
	assert.Contains(t, v2.UnqualifiedSearchRegistries, "registry.access.redhat.com")
}

func TestSetSystemRegistriesConfPath_PreservesV1BlockedAndInsecureRegistries(t *testing.T) {
	dir := t.TempDir()
	hostPath := filepath.Join(dir, "registries.conf")
	content := `[registries.search]
registries = ['registry.access.redhat.com']

[registries.block]
registries = ['docker.io', 'registry.hub.docker.com']

[registries.insecure]
registries = ['insecure-registry.example.com']
`
	require.NoError(t, os.WriteFile(hostPath, []byte(content), 0644))

	t.Setenv("CONTAINERS_REGISTRIES_CONF", hostPath)

	sys := &types.SystemContext{}
	require.NoError(t, SetSystemRegistriesConfPath(sys))
	assert.NotEqual(t, hostPath, sys.SystemRegistriesConfPath)
	assert.NotEmpty(t, sys.SystemRegistriesConfPath)

	ctx := &types.SystemContext{SystemRegistriesConfPath: sys.SystemRegistriesConfPath}

	reg, err := sysregistriesv2.FindRegistry(ctx, "docker.io/library/busybox")
	require.NoError(t, err)
	require.NotNil(t, reg)
	assert.True(t, reg.Blocked, "expected docker.io to be blocked")

	reg, err = sysregistriesv2.FindRegistry(ctx, "insecure-registry.example.com/app:latest")
	require.NoError(t, err)
	require.NotNil(t, reg)
	assert.True(t, reg.Insecure, "expected insecure-registry.example.com to be insecure")

	v2, err := sysregistriesv2.TryUpdatingCache(ctx)
	require.NoError(t, err)
	assert.Contains(t, v2.UnqualifiedSearchRegistries, "registry.access.redhat.com")
}

func TestSetSystemRegistriesConfPath_FallsBackToEmptyV2WhenV1ConfigIsEmpty(t *testing.T) {
	dir := t.TempDir()
	hostPath := filepath.Join(dir, "registries.conf")
	content := `[registries.search]
registries = []
`
	require.NoError(t, os.WriteFile(hostPath, []byte(content), 0644))

	t.Setenv("CONTAINERS_REGISTRIES_CONF", hostPath)

	sys := &types.SystemContext{}
	require.NoError(t, SetSystemRegistriesConfPath(sys))
	assert.NotEqual(t, hostPath, sys.SystemRegistriesConfPath)
	assert.NotEmpty(t, sys.SystemRegistriesConfPath)

	data, err := os.ReadFile(sys.SystemRegistriesConfPath)
	require.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(string(data)))
}

func TestSetSystemRegistriesConfPath_PreservesV1BlockedRegistryWithWhitespaceInHeader(t *testing.T) {
	dir := t.TempDir()
	hostPath := filepath.Join(dir, "registries.conf")
	// TOML allows whitespace around the dot in table headers.
	content := `[registries . block]
registries = ["docker.io"]
`
	require.NoError(t, os.WriteFile(hostPath, []byte(content), 0644))

	t.Setenv("CONTAINERS_REGISTRIES_CONF", hostPath)

	sys := &types.SystemContext{}
	require.NoError(t, SetSystemRegistriesConfPath(sys))
	assert.NotEqual(t, hostPath, sys.SystemRegistriesConfPath)
	assert.NotEmpty(t, sys.SystemRegistriesConfPath)

	ctx := &types.SystemContext{SystemRegistriesConfPath: sys.SystemRegistriesConfPath}
	reg, err := sysregistriesv2.FindRegistry(ctx, "docker.io/library/busybox")
	require.NoError(t, err)
	require.NotNil(t, reg)
	assert.True(t, reg.Blocked, "expected docker.io to be blocked")
}

func TestSetSystemRegistriesConfPath_ReturnsErrorWhenV1ConversionFails(t *testing.T) {
	dir := t.TempDir()
	hostPath := filepath.Join(dir, "registries.conf")
	// A v1 config with non-empty entries that cannot be parsed as valid registry locations.
	content := `[registries.block]
registries = ["https://invalid-scheme.example.com"]
`
	require.NoError(t, os.WriteFile(hostPath, []byte(content), 0644))

	t.Setenv("CONTAINERS_REGISTRIES_CONF", hostPath)

	sys := &types.SystemContext{}
	err := SetSystemRegistriesConfPath(sys)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert host v1 registries.conf to v2")
	assert.Empty(t, sys.SystemRegistriesConfPath)
}

func TestSetSystemRegistriesConfPath_ReturnsErrorWhenInvalidV2HasNoV1Markers(t *testing.T) {
	dir := t.TempDir()
	hostPath := filepath.Join(dir, "registries.conf")
	// A v2-looking file with a v2-only marker but invalid syntax (missing quote).
	content := `unqualified-search-registries = ["registry.access.redhat.com]
`
	require.NoError(t, os.WriteFile(hostPath, []byte(content), 0644))

	t.Setenv("CONTAINERS_REGISTRIES_CONF", hostPath)

	sys := &types.SystemContext{}
	err := SetSystemRegistriesConfPath(sys)
	require.Error(t, err)
	assert.Empty(t, sys.SystemRegistriesConfPath)
}

func TestSetSystemRegistriesConfPath_FallsBackWhenHostConfigMissing(t *testing.T) {
	hostPath := filepath.Join(t.TempDir(), "does-not-exist.conf")
	t.Setenv("CONTAINERS_REGISTRIES_CONF", hostPath)

	sys := &types.SystemContext{}
	require.NoError(t, SetSystemRegistriesConfPath(sys))
	assert.NotEmpty(t, sys.SystemRegistriesConfPath)
}

func TestSetSystemRegistriesConfPath_DoesNotCacheTransientErrors(t *testing.T) {
	dir := t.TempDir()
	hostPath := filepath.Join(dir, "registries.conf")

	// Create a malformed v2 file with no v1 markers.
	require.NoError(t, os.WriteFile(hostPath, []byte(`unqualified-search-registries = ["broken`), 0644))
	t.Setenv("CONTAINERS_REGISTRIES_CONF", hostPath)

	first := &types.SystemContext{}
	err := SetSystemRegistriesConfPath(first)
	require.Error(t, err)
	assert.Empty(t, first.SystemRegistriesConfPath)

	// Repair the file with a valid v2 config.
	require.NoError(t, os.WriteFile(hostPath, []byte(`unqualified-search-registries = ["registry.access.redhat.com"]
`), 0644))

	second := &types.SystemContext{}
	require.NoError(t, SetSystemRegistriesConfPath(second))
	assert.Equal(t, hostPath, second.SystemRegistriesConfPath)
}
