// Note: copied from: https://github.com/dexidp/dex/blob/ed920dc27ad79c3593037ad658552e8e80bab928/pkg/log/logger.go
package types

// Logger serves as an adapter interface for logger libraries
// so that dex does not depend on any of them directly.
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

func Deprecated(logger Logger, f string, args ...interface{}) {
	logger.Warnf("Deprecated: "+f, args...)
}
