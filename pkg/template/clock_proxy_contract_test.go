package template

import (
	"encoding/base64"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestSharedClockAndProxyContext(t *testing.T) {
	b, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)
	now, err := b.String(`repl{{ Now }}`)
	require.NoError(t, err)
	parsed, err := time.Parse(time.RFC3339, now)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), parsed, time.Minute)
	for name, env := range map[string]string{"HTTPProxy": "HTTP_PROXY", "HTTPSProxy": "HTTPS_PROXY", "NoProxy": "NO_PROXY"} {
		t.Setenv(env, "contract-value")
		actual, err := b.String("repl{{ " + name + " }}")
		require.NoError(t, err)
		require.Equal(t, "contract-value", actual)
	}
	encrypted, err := b.String(`repl{{ KubeSeal (TLSCACert "contract-seal" 1) "app" "name" "secret" }}`)
	require.NoError(t, err)
	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	require.NoError(t, err)
	require.Greater(t, len(decoded), 256)
}
