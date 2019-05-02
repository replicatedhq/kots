package logger

import (
	"fmt"
	"io"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-stack/stack"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
)

// New builds a logger from env using viper
func New(c *config.Config, out io.Writer) log.Logger {
	stdoutLogger := log.NewJSONLogger(out)
	stdoutLogger = log.With(stdoutLogger, "ts", log.DefaultTimestampUTC)
	stdoutLogger = log.With(stdoutLogger, "caller", pathCaller(5))
	stdoutLogger = withLevel(stdoutLogger, c.LogLevel)

	return stdoutLogger
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
		logger.Log("msg", "Unknown log level, using info", "received", lvl)
		return level.NewFilter(logger, level.AllowInfo())
	}
}

func pathCaller(depth int) log.Valuer {
	return func() interface{} {
		return fmt.Sprintf("%+s", stack.Caller(depth))
	}
}
