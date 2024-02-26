// Note: This is a modified version of: https://github.com/dexidp/dex/blob/ed920dc27ad79c3593037ad658552e8e80bab928/server/server.go
package types

// WebConfig holds the server's frontend templates and asset configuration.
type WebConfig struct {
	// Defaults to "( issuer URL )/theme/logo.png"
	LogoURL string

	// Defaults to "dex"
	Issuer string

	// Defaults to "light"
	Theme string
}

// ConnectorConfig is a configuration that can open a connector.
type ConnectorConfig interface {
}

// ConnectorsConfig variable provides an easy way to return a config struct
// depending on the connector type.
var ConnectorsConfig = map[string]func() ConnectorConfig{
	"oidc": func() ConnectorConfig { return new(OIDCConfig) },
}
