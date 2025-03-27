package upstream

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCachingReader(t *testing.T) {
	t.Run("empty read returns empty cached bytes", func(t *testing.T) {
		r := strings.NewReader("")
		cr := newCachingReader(r, 10)

		cachedBytes := cr.GetCachedBytes()
		require.Empty(t, cachedBytes)
	})

	t.Run("read smaller than cache size", func(t *testing.T) {
		data := "hello"
		r := strings.NewReader(data)
		cr := newCachingReader(r, 10)

		buf := make([]byte, 10)
		n, err := cr.Read(buf)

		require.NoError(t, err)
		require.Equal(t, 5, n)
		require.Equal(t, []byte("hello"), cr.GetCachedBytes())
	})

	t.Run("read exactly cache size", func(t *testing.T) {
		data := "helloworld"
		r := strings.NewReader(data)
		cr := newCachingReader(r, 10)

		buf := make([]byte, 10)
		n, err := cr.Read(buf)

		require.NoError(t, err)
		require.Equal(t, 10, n)
		require.Equal(t, []byte("helloworld"), cr.GetCachedBytes())
	})

	t.Run("read larger than cache size", func(t *testing.T) {
		data := "hello world this is a test"
		r := strings.NewReader(data)
		cr := newCachingReader(r, 10)

		buf := make([]byte, len(data))
		n, err := cr.Read(buf)

		require.NoError(t, err)
		require.Equal(t, len(data), n)
		// Should only have last 10 bytes
		require.Equal(t, []byte(" is a test"), cr.GetCachedBytes())
	})

	t.Run("multiple reads accumulate in cache", func(t *testing.T) {
		r := strings.NewReader("first second third")
		cr := newCachingReader(r, 10)

		buf1 := make([]byte, 5) // "first"
		n1, err := cr.Read(buf1)
		require.NoError(t, err)
		require.Equal(t, 5, n1)

		buf2 := make([]byte, 7) // " second"
		n2, err := cr.Read(buf2)
		require.NoError(t, err)
		require.Equal(t, 7, n2)

		// Cache should now have "rst second"
		require.Equal(t, []byte("rst second"), cr.GetCachedBytes())

		buf3 := make([]byte, 6) // " third"
		n3, err := cr.Read(buf3)
		require.NoError(t, err)
		require.Equal(t, 6, n3)

		// Cache should now have "second third"
		require.Equal(t, []byte("cond third"), cr.GetCachedBytes())
	})

	t.Run("cache wraps around correctly", func(t *testing.T) {
		r := strings.NewReader("abcdefghijklmnopqrstuvwxyz")
		cr := newCachingReader(r, 10)

		buf := make([]byte, 8)

		// Read "abcdefgh"
		n, err := cr.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 8, n)
		require.Equal(t, []byte("abcdefgh"), cr.GetCachedBytes())

		// Read "ijklmnop"
		n, err = cr.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 8, n)

		// Cache should now contain "defghijklmnop", but truncated to 10: "ghijklmnop"
		require.Equal(t, []byte("ghijklmnop"), cr.GetCachedBytes())
	})

	t.Run("read until EOF", func(t *testing.T) {
		data := "test data for EOF testing"
		r := strings.NewReader(data)
		cr := newCachingReader(r, 10)

		buf := make([]byte, 100) // Larger than data
		n, err := cr.Read(buf)

		require.NoError(t, err)
		require.Equal(t, len(data), n)

		// Try to read again, should get EOF
		n, err = cr.Read(buf)
		require.Equal(t, 0, n)
		require.Equal(t, io.EOF, err)

		// Last 10 bytes should be in cache
		require.Equal(t, []byte("OF testing"), cr.GetCachedBytes())
	})

	t.Run("large reads with multiple chunks", func(t *testing.T) {
		// Create a 100-byte data source
		var data bytes.Buffer
		for i := 0; i < 100; i++ {
			data.WriteByte(byte(i%26 + 'a'))
		}

		r := bytes.NewReader(data.Bytes())
		cr := newCachingReader(r, 15)

		// Read in chunks of 30 bytes
		buf := make([]byte, 30)

		// First read: 0-29
		n, err := cr.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 30, n)

		// Second read: 30-59
		n, err = cr.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 30, n)

		// Third read: 60-89
		n, err = cr.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 30, n)

		// Last read: 90-99
		n, err = cr.Read(buf)
		require.NoError(t, err)
		require.Equal(t, 10, n)

		// Cache should contain the last 15 bytes (86-100)
		expected := make([]byte, 15)
		for i := 0; i < 15; i++ {
			expected[i] = byte((i+85)%26 + 'a')
		}

		require.Equal(t, expected, cr.GetCachedBytes())
	})
}
