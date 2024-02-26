// Note: This is a modified version of: https://github.com/dexidp/dex/blob/ed920dc27ad79c3593037ad658552e8e80bab928/connector/oidc/oidc.go
// Package oidc implements logging in through OpenID Connect providers.
package types

// OIDCConfig holds configuration options for OpenID Connect logins.
type OIDCConfig struct {
	Issuer       string `json:"issuer"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	RedirectURI  string `json:"redirectURI"`

	Scopes []string `json:"scopes"` // defaults to "profile" and "email"

	// Override the value of email_verified to true in the returned claims
	InsecureSkipEmailVerified bool `json:"insecureSkipEmailVerified"`

	// InsecureEnableGroups enables groups claims. This is disabled by default until https://github.com/dexidp/dex/issues/1065 is resolved
	InsecureEnableGroups bool `json:"insecureEnableGroups"`

	// Disable certificate verification
	InsecureSkipVerify bool `json:"insecureSkipVerify"`

	// GetUserInfo uses the userinfo endpoint to get additional claims for
	// the token. This is especially useful where upstreams return "thin"
	// id tokens
	GetUserInfo bool `json:"getUserInfo"`

	UserIDKey string `json:"userIDKey"`

	UserNameKey string `json:"userNameKey"`

	// PromptType will be used fot the prompt parameter (when offline_access, by default prompt=consent)
	PromptType string `json:"promptType"`

	ClaimMapping struct {
		// Configurable key which contains the preferred username claims
		PreferredUsernameKey string `json:"preferred_username"` // defaults to "preferred_username"

		// Configurable key which contains the email claims
		EmailKey string `json:"email"` // defaults to "email"

		// Configurable key which contains the groups claims
		GroupsKey string `json:"groups"` // defaults to "groups"
	} `json:"claimMapping"`
}
