package types

import "time"

type Session struct {
	ID        string
	IssuedAt  time.Time
	ExpiresAt time.Time
	Roles     []string
	HasRBAC   bool
}
