package types

import "time"

type Session struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt time.Time
}
