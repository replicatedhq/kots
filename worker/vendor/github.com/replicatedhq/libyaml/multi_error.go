package libyaml

import (
	"fmt"
)

type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	return fmt.Sprintf("%q", e.Errors)
}

func (e *MultiError) Append(err error) {
	if err == nil {
		return
	}
	e.Errors = append(e.Errors, err)
}

func (e *MultiError) ErrorOrNil() error {
	if e == nil || len(e.Errors) == 0 {
		return nil
	}
	return e
}
