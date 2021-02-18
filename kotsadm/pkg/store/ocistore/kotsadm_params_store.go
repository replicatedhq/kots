package ocistore

func (s OCIStore) IsKotsadmIDGenerated() (bool, error) {
	return false, ErrNotImplemented
}

func (s OCIStore) SetIsKotsadmIDGenerated() error {
	return ErrNotImplemented
}
