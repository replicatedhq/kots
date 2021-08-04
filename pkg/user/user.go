package user

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	usertypes "github.com/replicatedhq/kots/pkg/user/types"
	"golang.org/x/crypto/bcrypt"
)

var (
	loginMutex         sync.Mutex
	ErrInvalidPassword = errors.New("invalid password")
	ErrTooManyAttempts = errors.New("too many attempts")
)

func LogIn(password string) (*usertypes.User, error) {
	loginMutex.Lock()
	defer loginMutex.Unlock()

	shaBytes, err := store.GetStore().GetSharedPasswordBcrypt()

	// this is rough...  the error is defined twice but we can't wrap it if this is the error
	if err != nil && err.Error() == ErrTooManyAttempts.Error() {
		return nil, ErrTooManyAttempts
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to get shared password bcrypt")
	}

	if err := bcrypt.CompareHashAndPassword(shaBytes, []byte(password)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			if err := store.GetStore().FlagInvalidPassword(); err != nil {
				logger.Infof("failed to flag failed login: %v", err)
			}
			return nil, ErrInvalidPassword
		}

		return nil, errors.Wrap(err, "failed to compare password")
	}

	if err := store.GetStore().FlagSuccessfulLogin(); err != nil {
		logger.Error(errors.Wrap(err, "failed to flag successful login"))
	}

	return &usertypes.User{
		ID: "000000",
	}, nil
}
