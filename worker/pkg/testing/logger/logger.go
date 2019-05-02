package logger

import (
	"github.com/go-kit/kit/log"
)

var _ log.Logger = &TestLogger{}

type TestingT interface {
	Log(...interface{})
}

type TestLogger struct {
	T TestingT
}

func (t TestLogger) Log(keyvals ...interface{}) error {
	t.T.Log(keyvals...)
	return nil
}
