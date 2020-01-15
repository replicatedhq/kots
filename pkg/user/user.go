package user

import (
	"os"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID string
}

func LogIn(password string) (*User, error) {
	if err := bcrypt.CompareHashAndPassword([]byte(os.Getenv("SHARED_PASSWORD_BCRYPT")), []byte(password)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to compare password")
	}

	return &User{
		ID: "000000",
	}, nil
}
