package s3pg

type scannable interface {
	Scan(dest ...interface{}) error
}
