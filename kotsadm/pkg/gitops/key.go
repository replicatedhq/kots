package gitops

import (
	"crypto/rsa"
	"math/rand"
	"sync"
	"time"
)

var (
	// Generating the gitops key is very slow (~10s).
	// Pre-seed it to save time in the UI.
	preseedPrivateKey   *rsa.PrivateKey
	preseedPrivateKeyMu sync.Mutex
)

var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

	go generatePreseedPrivateKey()
}

func getPrivateKey() (*rsa.PrivateKey, error) {
	preseedPrivateKeyMu.Lock()
	key := preseedPrivateKey
	preseedPrivateKey = nil // one time use
	preseedPrivateKeyMu.Unlock()

	go generatePreseedPrivateKey()

	if key != nil {
		return key, nil
	}
	return rsa.GenerateKey(r, 4096)
}

func generatePreseedPrivateKey() {
	preseedPrivateKeyMu.Lock()
	defer preseedPrivateKeyMu.Unlock()
	preseedPrivateKey, _ = rsa.GenerateKey(r, 4096)
}
