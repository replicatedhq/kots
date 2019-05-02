package logger

import (
	"fmt"
	golog "log"
	"os"

	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-stack/stack"
)

var (
	fullPathCaller = pathCaller(3)
	globalLogger   log.Logger
	logMtx         sync.Mutex
)

// FromEnv constructs a new logger using environment
func FromEnv() log.Logger {

	// one at a time plz
	logMtx.Lock()
	defer logMtx.Unlock()

	if globalLogger != nil {
		return globalLogger
	}

	globalLogger = withFormat(os.Getenv("log-format"))
	globalLogger = log.With(globalLogger, "ts", log.DefaultTimestampUTC)
	globalLogger = withLevel(globalLogger, os.Getenv("log-level"))
	globalLogger = log.With(globalLogger, "caller", fullPathCaller)
	golog.SetOutput(log.NewStdlibAdapter(level.Debug(globalLogger)))
	return globalLogger
}

func withFormat(format string) log.Logger {
	switch format {
	case "json":
		return log.NewJSONLogger(os.Stdout)
	case "logfmt":
		return log.NewLogfmtLogger(os.Stdout)
	default:
		return log.NewLogfmtLogger(os.Stdout)
	}

}

func withLevel(logger log.Logger, lvl string) log.Logger {
	switch lvl {
	case "debug":
		return level.NewFilter(logger, level.AllowDebug())
	case "info":
		return level.NewFilter(logger, level.AllowInfo())
	case "warn":
		return level.NewFilter(logger, level.AllowWarn())
	case "error":
		return level.NewFilter(logger, level.AllowError())
	case "off":
		return level.NewFilter(logger, level.AllowNone())
	default:
		logger.Log("msg", "Unknown log level, using debug", "received", lvl)
		return level.NewFilter(logger, level.AllowDebug())
	}
}

func pathCaller(depth int) log.Valuer {
	return func() interface{} {
		return fmt.Sprintf("%+s", stack.Caller(depth))
	}
}
