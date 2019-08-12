package logger

import (
	"io"

	"github.com/replicatedhq/kotsadm/worker/pkg/config"
	"go.uber.org/zap"
)

func New(c *config.Config, out io.Writer) *zap.SugaredLogger {
	if c.LogLevel == "debug" {
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		return logger.Sugar()
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	return logger.Sugar()
}
