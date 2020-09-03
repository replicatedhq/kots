package gitops

import (
	"crypto/rand"
	"crypto/rsa"
	"sync"
)

var (
	// Generating the gitops key is very slow (~10s).
	// Pre-seed it to save time in the UI.
	preseedPrivateKey   *rsa.PrivateKey
	preseedPrivateKeyMu sync.Mutex
)

func init() {
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
	return rsa.GenerateKey(rand.Reader, 4096)
}

func generatePreseedPrivateKey() {
	preseedPrivateKeyMu.Lock()
	defer preseedPrivateKeyMu.Unlock()
	preseedPrivateKey, _ = rsa.GenerateKey(rand.Reader, 4096)
}
