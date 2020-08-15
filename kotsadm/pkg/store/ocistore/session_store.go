package ocistore

import (
	"github.com/pkg/errors"
	sessiontypes "github.com/replicatedhq/kots/kotsadm/pkg/session/types"
	usertypes "github.com/replicatedhq/kots/kotsadm/pkg/user/types"
)

func (s OCIStore) CreateSession(forUser *usertypes.User) (*sessiontypes.Session, error) {
	return nil, errors.New("not implemented")
}

func (s OCIStore) GetSession(id string) (*sessiontypes.Session, error) {
	return nil, errors.New("not implemented")
}

func (s OCIStore) DeleteSession(id string) error {
	return errors.New("not implemented")
}
