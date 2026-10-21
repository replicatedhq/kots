package template

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSharedClockAndProxyContext(t *testing.T) {
	b, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)
	now, err := b.String(`repl{{ Now }}`)
	require.NoError(t, err)
	parsed, err := time.Parse(time.RFC3339, now)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), parsed, time.Minute)
	tests := []struct{ name, function, env string }{
		{"HTTP proxy", "HTTPProxy", "HTTP_PROXY"},
		{"HTTPS proxy", "HTTPSProxy", "HTTPS_PROXY"},
		{"proxy exclusions", "NoProxy", "NO_PROXY"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.env, "contract-value")
			actual, err := b.String("repl{{ " + test.function + " }}")
			require.NoError(t, err)
			require.Equal(t, "contract-value", actual)
		})
	}
	encrypted, err := b.String(`repl{{ KubeSeal (TLSCACert "contract-seal" 1) "app" "name" "secret" }}`)
	require.NoError(t, err)
	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	require.NoError(t, err)
	require.Greater(t, len(decoded), 256)
}
