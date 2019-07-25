package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CommonSlicePrefix(t *testing.T) {
	tests := []struct {
		name     string
		first    []string
		second   []string
		expected []string
	}{
		{
			name:     "no common",
			first:    []string{"a", "b"},
			second:   []string{"1", "2"},
			expected: []string{},
		},
		{
			name:     "partial",
			first:    []string{"1", "2", "3"},
			second:   []string{"1", "a", "b"},
			expected: []string{"1"},
		},
		{
			name:     "exact",
			first:    []string{"l", "m", "n", "o"},
			second:   []string{"l", "m", "n", "o"},
			expected: []string{"l", "m", "n", "o"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			common := CommonSlicePrefix(test.first, test.second)
			assert.Equal(t, test.expected, common)
		})
	}
}
