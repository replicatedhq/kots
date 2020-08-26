package rand

import (
	"math/rand"
	"time"
)

const LOWER_CASE = "abcdefghijklmnopqrstuvwxyz"
const UPPER_CASE = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const NUMERIC = "1234567890"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
