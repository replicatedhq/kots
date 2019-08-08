package applier

import (
	"bufio"
	"os/exec"

	"github.com/pkg/errors"
)

func Run(cmd *exec.Cmd, stdoutChan *chan []byte, stderrChan *chan []byte) error {
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrapf(err, "failed to create stdout reader")
	}
	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrapf(err, "failed to create stderr reader")
	}

	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "failed to start commmand")
	}

	if stdoutChan != nil {
		stdoutScanner := bufio.NewScanner(stdoutReader)
		stdoutScanner.Split(bufio.ScanLines)
		for stdoutScanner.Scan() {
			*stdoutChan <- stdoutScanner.Bytes()
		}
	}

	if stderrChan != nil {
		stderrScanner := bufio.NewScanner(stderrReader)
		stderrScanner.Split(bufio.ScanLines)
		for stderrScanner.Scan() {
			*stderrChan <- stderrScanner.Bytes()
		}
	}

	err = cmd.Wait()
	return err
}
