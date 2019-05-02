package warnings

import (
	"fmt"

	"github.com/pkg/errors"
)

// WarnShouldUseUpdate is the message printed to the user when they attempt
// to use "ship init" with a present state file on disk
var WarnShouldUseUpdate = warning{msg: `To build on your progress, run "ship update"`}

var WarnCannotRemoveState = warning{msg: `Existing state was found that Ship cannot automatically remove. Please delete the existing state and try again.`}

// WarnShouldMoveDirectory is the message printed to the user when they attempt to run ship when files like `base` or
// `overlays` are already present
func WarnShouldMoveDirectory(dir string) error {
	return warning{msg: fmt.Sprintf(`Found existing directory %q. To avoid losing work, please move or remove %q before proceeding, or re-run with --rm-asset-dest.`, dir, dir)}
}

func WarnFileNotFound(filePath string) error {
	return warning{msg: fmt.Sprintf(`File %q was not found.`, filePath)}
}

type warning struct {
	msg string
}

func (w warning) Error() string {
	return w.msg
}

func IsWarning(err error) bool {
	cause := errors.Cause(err)
	_, ok := cause.(warning)
	return ok
}

func StripStackIfWarning(err error) error {
	if IsWarning(err) {
		return errors.Cause(err)
	}
	return err
}
