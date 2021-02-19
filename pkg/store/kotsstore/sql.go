package kotsstore

type scannable interface {
	Scan(dest ...interface{}) error
}
