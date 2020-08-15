package types

import (
	"time"
)

type RedactorList struct {
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Created     time.Time `json:"createdAt"`
	Updated     time.Time `json:"updatedAt"`
	Enabled     bool      `json:"enabled"`
	Description string    `json:"description"`
}
