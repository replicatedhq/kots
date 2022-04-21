package user

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/store"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNewPasswordTooShort          = errors.New("new password must be at least 6 characters")
	ErrNewPasswordShouldBeDifferent = errors.New("new password should be different from current password")
)

// ChangePassword - will compare the current password with the stored password and change to new password if they match
func ChangePassword(kotsStore store.Store, currentPassword, newPassword string) error {
	if err := validatePassword(kotsStore, currentPassword); err != nil {
		return err
	}

	shaBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return errors.Wrap(err, "failed to generate new encrypted password")
	}

	if err := kotsStore.SetSharedPasswordBcrypt(shaBytes); err != nil {
		return errors.Wrap(err, "failed to set new shared password bcrypt")
	}

	return nil
}

// validatePassword - will compare the password with the stored password and return an error if they don't match
func validatePassword(kotsStore store.Store, currentPassword string) error {
	shaBytes, err := kotsStore.GetSharedPasswordBcrypt()
	if err != nil {
		return errors.Wrap(err, "failed to get current shared password bcrypt")
	}

	if err := bcrypt.CompareHashAndPassword(shaBytes, []byte(currentPassword)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return ErrInvalidPassword
		}

		return errors.Wrap(err, "failed to compare current password")
	}

	return nil
}

// ValidatePasswordInput - will validate length and complexity of new password and check if it is different from current password
func ValidatePasswordInput(currentPassword string, newPassword string) error {
	if len(newPassword) < 6 {
		return ErrNewPasswordTooShort
	}

	if newPassword == currentPassword {
		return ErrNewPasswordShouldBeDifferent
	}
	return nil
}
