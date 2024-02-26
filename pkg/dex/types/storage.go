// Note: This is a modified version of: https://github.com/dexidp/dex/blob/ed920dc27ad79c3593037ad658552e8e80bab928/storage/storage.go
package types

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// StorageClient represents an OAuth2 client.
//
// For further reading see:
//   - Trusted peers: https://developers.google.com/identity/protocols/CrossClientAuth
//   - Public clients: https://developers.google.com/api-client-library/python/auth/installed-app
type StorageClient struct {
	// Client ID and secret used to identify the client.
	ID        string `json:"id" yaml:"id"`
	IDEnv     string `json:"idEnv" yaml:"idEnv"`
	Secret    string `json:"secret" yaml:"secret"`
	SecretEnv string `json:"secretEnv" yaml:"secretEnv"`

	// A registered set of redirect URIs. When redirecting from dex to the client, the URI
	// requested to redirect to MUST match one of these values, unless the client is "public".
	RedirectURIs []string `json:"redirectURIs" yaml:"redirectURIs"`

	// Name and LogoURL used when displaying this client to the end user.
	Name    string `json:"name" yaml:"name"`
	LogoURL string `json:"logoURL" yaml:"logoURL"`
}

// StoragePassword is an email to password mapping managed by the storage.
type StoragePassword struct {
	// Email and identifying name of the password. Emails are assumed to be valid and
	// determining that an end-user controls the address is left to an outside application.
	//
	// Emails are case insensitive and should be standardized by the storage.
	//
	// Storages that don't support an extended character set for IDs, such as '.' and '@'
	// (cough cough, kubernetes), must map this value appropriately.
	Email string `json:"email"`

	// Bcrypt encoded hash of the password. This package enforces a min cost value of 10
	Hash []byte `json:"hash"`

	// Optional username to display. NOT used during login.
	Username string `json:"username"`

	// Randomly generated user ID. This is NOT the primary ID of the Password object.
	UserID string `json:"userID"`
}

func (p *StoragePassword) UnmarshalJSON(b []byte) error {
	var data struct {
		Email       string `json:"email"`
		Username    string `json:"username"`
		UserID      string `json:"userID"`
		Hash        string `json:"hash"`
		HashFromEnv string `json:"hashFromEnv"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	*p = StoragePassword{
		Email:    data.Email,
		Username: data.Username,
		UserID:   data.UserID,
	}
	if len(data.Hash) == 0 && len(data.HashFromEnv) > 0 {
		data.Hash = os.Getenv(data.HashFromEnv)
	}
	if len(data.Hash) == 0 {
		return fmt.Errorf("no password hash provided")
	}

	// If this value is a valid bcrypt, use it.
	_, bcryptErr := bcrypt.Cost([]byte(data.Hash))
	if bcryptErr == nil {
		p.Hash = []byte(data.Hash)
		return nil
	}

	// For backwards compatibility try to base64 decode this value.
	hashBytes, err := base64.StdEncoding.DecodeString(data.Hash)
	if err != nil {
		return fmt.Errorf("malformed bcrypt hash: %v", bcryptErr)
	}
	if _, err := bcrypt.Cost(hashBytes); err != nil {
		return fmt.Errorf("malformed bcrypt hash: %v", err)
	}
	p.Hash = hashBytes
	return nil
}
