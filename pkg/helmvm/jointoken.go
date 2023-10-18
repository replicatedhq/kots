package helmvm

import (
	"encoding/base64"
	"encoding/json"

	"github.com/google/uuid"
)

// joinToken is a struct that holds both the actual token and the cluster id. This is marshaled
// and base64 encoded and used as argument to the join command in the other nodes.
type joinToken struct {
	ClusterID uuid.UUID `json:"clusterID"`
	Token     string    `json:"token"`
	Role      string    `json:"role"`
}

// Encode encodes a JoinToken to base64.
func (j *joinToken) Encode() (string, error) {
	b, err := json.Marshal(j)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
