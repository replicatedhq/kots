package store

type SQLiteStore struct {
	dbFilename string
}

func (s *SQLiteStore) DatabaseUserExists(user string) (bool, error) {
	// SQLite has no notion of db users
	return true, nil
}

func (s *SQLiteStore) CreateDexDatabase(database string, user string, password string) error {
	// SQLite database is a file on disk that does not need to be created ahead of time
	return nil
}
