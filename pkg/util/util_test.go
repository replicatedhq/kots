package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func Test_SplitStringOnLen(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		max      int
		expected []string
	}{
		{
			name:     "single part",
			in:       "this is a test",
			max:      1000,
			expected: []string{"this is a test"},
		},
		{
			name:     "even parts",
			in:       "fourfivenine",
			max:      4,
			expected: []string{"four", "five", "nine"},
		},
		{
			name:     "too big",
			in:       "one two six",
			max:      7,
			expected: []string{"one two", " six"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			parts, err := SplitStringOnLen(test.in, test.max)
			req.NoError(err)

			assert.Equal(t, test.expected, parts)
		})
	}
}
