package types

type PasswordUser struct {
	ID string

	Email string
}

func (u PasswordUser) GetID() string {
	return u.ID
}

func (u PasswordUser) GetUsername() string {
	return u.Email
}
