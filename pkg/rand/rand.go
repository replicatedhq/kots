package rand

import (
	"math/rand"
	"time"
)

const LOWER_CASE = "abcdefghijklmnopqrstuvwxyz"
const UPPER_CASE = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const NUMERIC = "1234567890"

var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
}

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}
