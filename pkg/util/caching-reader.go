package util

import (
	"io"
)

// CachingReader is a reader that caches the last N bytes read
// to help with debugging when encountering errors in tar or gzip extraction
type CachingReader struct {
	r        io.Reader
	buf      []byte
	size     int
	position int
	full     bool
}

// NewCachingReader creates a new caching reader that caches the last size bytes
func NewCachingReader(r io.Reader, size int) *CachingReader {
	return &CachingReader{
		r:    r,
		buf:  make([]byte, size),
		size: size,
	}
}

// Read implements io.Reader
func (c *CachingReader) Read(p []byte) (n int, err error) {
	n, err = c.r.Read(p)
	if n > 0 {
		// Cache the bytes read
		if n <= c.size {
			// If we're reading fewer bytes than our buffer size
			if c.position+n <= c.size {
				// If we can append without wrapping
				copy(c.buf[c.position:], p[:n])
				c.position += n
			} else {
				// Need to wrap around
				firstPart := c.size - c.position
				copy(c.buf[c.position:], p[:firstPart])
				copy(c.buf[:n-firstPart], p[firstPart:n])
				c.position = n - firstPart
				c.full = true
			}
		} else {
			// If reading more bytes than our buffer size, just take the last c.size bytes
			copy(c.buf, p[n-c.size:n])
			c.position = 0
			c.full = true
		}
	}
	return n, err
}

// GetCachedBytes returns the last bytes read up to size
func (c *CachingReader) GetCachedBytes() []byte {
	if !c.full && c.position == 0 {
		return []byte{}
	}

	if !c.full {
		return c.buf[:c.position]
	}

	result := make([]byte, c.size)

	// Copy from position to end
	copy(result, c.buf[c.position:])
	// Copy from start to position
	copy(result[c.size-c.position:], c.buf[:c.position])

	return result
}
