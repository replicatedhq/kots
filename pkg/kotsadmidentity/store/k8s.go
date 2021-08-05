package store

type K8sStore struct {
}

func (s *K8sStore) DatabaseUserExists(user string) (bool, error) {
	return true, nil
}

func (s *K8sStore) CreateDexDatabase(database string, user string, password string) error {
	return nil
}
