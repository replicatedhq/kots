package applier

import (
	"io"
	"os/exec"
	"sync"

	"github.com/pkg/errors"
)

func Run(cmd *exec.Cmd) ([]byte, []byte, error) {
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create stdout reader")
	}
	defer stdoutReader.Close()

	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create stderr reader")
	}
	defer stderrReader.Close()

	err = cmd.Start()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to start commmand")
	}

	var stdout, stderr []byte
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		stdout, _ = io.ReadAll(stdoutReader)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		stderr, _ = io.ReadAll(stderrReader)
	}()

	// cmd.Wait() must be called after all readers have completed
	wg.Wait()

	err = cmd.Wait()
	return stdout, stderr, err
}
