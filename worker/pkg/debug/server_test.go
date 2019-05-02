package debug

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerGetHealthz(t *testing.T) {
	tests := []struct {
		name   string
		status bool
	}{
		{
			name:   "1 ok",
			status: true,
		},
		{
			name:   "2 not ok",
			status: false,
		},
		{
			name:   "3 ok",
			status: true,
		},
	}

	addr := getServerAddr(t)
	server := &http.Server{Addr: addr, Handler: NewServer(log.NewNopLogger())}
	defer server.Close()
	go func() {
		err := server.ListenAndServe()
		require.NoError(t, err)
	}()

	Init()

	time.Sleep(time.Millisecond * 10) // wait for server

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetStatus(tt.status)

			resp, err := http.Get(fmt.Sprintf("http://%s/healthz", addr))
			require.NoError(t, err)
			defer resp.Body.Close()

			dec := json.NewDecoder(resp.Body)
			var out map[string]interface{}
			err = dec.Decode(&out)
			require.NoError(t, err)

			assert.Equal(t, strconv.FormatBool(tt.status), out["ok"])
		})
	}
}

func getServerAddr(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ln.Close()
	return ln.Addr().String()
}
