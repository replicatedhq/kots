package cursor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Comparable(t *testing.T) {
	tests := []struct {
		name    string
		c1      string
		c1Error bool
		c2      string
		c2Error bool

		isComparable bool
		isEqual      bool
		isBefore     bool
		isAfter      bool
	}{
		{
			name:         "comparable and equal",
			c1:           "10",
			c2:           "10",
			isEqual:      true,
			isComparable: true,
		},
		{
			name:         "comparable and before",
			c1:           "1",
			c2:           "10",
			isBefore:     true,
			isComparable: true,
		},
		{
			name:         "comparable and after",
			c1:           "100",
			c2:           "10",
			isAfter:      true,
			isComparable: true,
		},
		{
			name:    "not valid cursor",
			c1:      "1.0.4",
			c2:      "10",
			c1Error: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c1, err := NewCursor(test.c1)
			if test.c1Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			c2, err := NewCursor(test.c2)
			if test.c2Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if c1 == nil || c2 == nil {
				return
			}

			assert.Equal(t, test.isComparable, c1.Comparable(c2))
			assert.Equal(t, test.isEqual, c1.Equal(c2))
			assert.Equal(t, test.isBefore, c1.Before(c2))
			assert.Equal(t, test.isAfter, c1.After(c2))
		})
	}
}
