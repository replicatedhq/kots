package replicated

import (
	"fmt"

	"github.com/pkg/errors"
)

func RunIntegration() error {
	fmt.Println("Running replicated tests")

	if err := runPullTests(); err != nil {
		return errors.Wrap(err, "failed to run pull tests")
	}

	if err := runInstallTests(); err != nil {
		return errors.Wrap(err, "failed to run install tests")
	}

	return nil
}
