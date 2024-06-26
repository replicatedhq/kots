// Note: This is a modified version of: https://github.com/dexidp/dex/blob/ed920dc27ad79c3593037ad658552e8e80bab928/storage/storage.go
package types

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
