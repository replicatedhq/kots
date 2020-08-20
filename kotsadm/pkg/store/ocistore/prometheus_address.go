package ocistore

func (s OCIStore) GetPrometheusAddress() (string, error) {
	return "", ErrNotImplemented
}

func (s OCIStore) SetPrometheusAddress(address string) error {
	return ErrNotImplemented
}
