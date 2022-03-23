package cursor

import (
	"strconv"

	"github.com/pkg/errors"
)

type Cursor interface {
	Comparable(Cursor) bool
	Equal(Cursor) bool
	Before(Cursor) bool
	After(Cursor) bool
}

type SequenceCursor struct {
	cursor uint64
}

func MustParse(s string) Cursor {
	if c, err := strconv.ParseUint(s, 10, 64); err == nil {
		return SequenceCursor{cursor: c}
	}
	panic(errors.Errorf("cannot use %q to construct cursor", s))
}

func NewCursor(s string) (Cursor, error) {
	if c, err := strconv.ParseUint(s, 10, 64); err == nil {
		return SequenceCursor{cursor: c}, nil
	}
	return nil, errors.Errorf("cannot use %q to construct cursor", s)
}

func (c SequenceCursor) Comparable(o Cursor) bool {
	switch o.(type) {
	case SequenceCursor, *SequenceCursor:
		return true
	default:
		return false
	}
}

func (c SequenceCursor) Equal(o Cursor) bool {
	return c.cursor == o.(SequenceCursor).cursor
}

func (c SequenceCursor) Before(o Cursor) bool {
	return c.cursor < o.(SequenceCursor).cursor
}

func (c SequenceCursor) After(o Cursor) bool {
	return c.cursor > o.(SequenceCursor).cursor
}
