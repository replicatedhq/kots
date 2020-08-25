package apiserver

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
)

func bootstrap() error {
	if err := store.GetStore().Init(); err != nil {
		return errors.Wrap(err, "failed to init store")
	}

	return nil
}
