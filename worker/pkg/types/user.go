package types

// User is a shared interface for the different user types
type User interface {
	GetID() string
	GetUsername() string
}
