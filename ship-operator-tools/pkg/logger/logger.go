package logger

import (
	"fmt"
	golog "log"
	"os"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-stack/stack"
	"github.com/spf13/viper"
)

var (
	fullPathCaller = pathCaller(3)
	globalLogger   log.Logger
	logMtx         sync.Mutex
)

// New builds a logger from env using viper
func New(v *viper.Viper) log.Logger {

	fullPathCaller := pathCaller(3)
	var stdoutLogger log.Logger
	stdoutLogger = withFormat(viper.GetString("log-format"))
	stdoutLogger = log.With(stdoutLogger, "ts", log.DefaultTimestampUTC)
	stdoutLogger = log.With(stdoutLogger, "caller", fullPathCaller)
	stdoutLogger = withLevel(stdoutLogger, v.GetString("log-level"))

	golog.SetOutput(log.NewStdlibAdapter(level.Debug(stdoutLogger)))
	return stdoutLogger
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
