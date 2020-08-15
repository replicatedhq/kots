package ocistore

import (
	"github.com/pkg/errors"
)

func (s OCIStore) GetPrometheusAddress() (string, error) {
	return "", errors.New("not implemented")
}

func (s OCIStore) SetPrometheusAddress(address string) error {
	return errors.New("not implemented")
}
