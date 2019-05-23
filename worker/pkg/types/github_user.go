package types

type GitHubUser struct {
	ID string

	Username string
}

func (u GitHubUser) GetID() string {
	return u.ID
}

func (u GitHubUser) GetUsername() string {
	return u.Username
}
