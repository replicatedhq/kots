package types

import "time"

type Session struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt time.Time
	Roles     []SessionRole
	HasRBAC   bool
}

type SessionRole struct {
	ID       string
	Policies []SessionPolicy
}

type SessionPolicy struct {
	ID      string
	Allowed []string
	Denied  []string
}
