package registry

import (
	"encoding/json"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
)

// Copied from github.com/containers/image/docker
// Time fields aren't used
type BearerToken struct {
	Token          string    `json:"token"`
	AccessToken    string    `json:"access_token"`
	ExpiresIn      int       `json:"expires_in"`
	IssuedAt       time.Time `json:"issued_at"`
	expirationTime time.Time
}

func newBearerTokenFromJSONBlob(blob []byte) (*BearerToken, error) {
	token := new(BearerToken)
	if err := json.Unmarshal(blob, &token); err != nil {
		return nil, err
	}
	if token.Token == "" {
		token.Token = token.AccessToken
	}
	return token, nil
}

type accessItem struct {
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

type claims struct {
	jwt.StandardClaims `json:",inline"`
	Access             []accessItem `json:"access"`
}

func (t *BearerToken) getJwtToken() (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(t.Token, &claims{}, nil)
	if err != nil {
		if err, ok := err.(*jwt.ValidationError); ok {
			// Since we don't have the key used to issue the token, we have to skip verification.
			if err.Errors == jwt.ValidationErrorUnverifiable {
				return token, nil
			}
		}
		return nil, errors.Wrap(err, "failed to parse token with claims")
	}

	// this should never happen because of the missing key
	return token, nil
}

func getJwtTokenClaims(token *jwt.Token) (*claims, error) {
	tokenClaims, ok := token.Claims.(*claims)
	if !ok {
		return nil, errors.Errorf("unsupported claims type %T", token.Claims)
	}
	return tokenClaims, nil
}
