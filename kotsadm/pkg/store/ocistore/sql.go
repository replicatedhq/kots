package ocistore

type scannable interface {
	Scan(dest ...interface{}) error
}
