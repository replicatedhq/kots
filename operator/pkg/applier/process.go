package applier

import (
	"io/ioutil"
	"os/exec"

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

	stdout, _ := ioutil.ReadAll(stdoutReader)
	stderr, _ := ioutil.ReadAll(stderrReader)

	err = cmd.Wait()
	return stdout, stderr, err
}
