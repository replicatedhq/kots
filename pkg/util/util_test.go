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

func TestIntPointer(t *testing.T) {
	tests := []struct {
		name string
		x    int
		want int64
	}{
		{
			name: "zero",
			x:    0,
			want: int64(0),
		},
		{
			name: "positive",
			x:    100,
			want: int64(100),
		},
		{
			name: "negative",
			x:    -128,
			want: int64(-128),
		},
		{
			name: "int max",
			x:    1<<31 - 1,
			want: int64(1<<31 - 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got := IntPointer(tt.x)
			req.Equal(tt.want, *got)
		})
	}
}

func TestGenPassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "8",
			length: 8,
		},
		{
			name:   "32",
			length: 32,
		},
		{
			name:   "0",
			length: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got := GenPassword(tt.length)
			req.Len(got, tt.length)
		})
	}
}

func TestCompareStringArrays(t *testing.T) {
	tests := []struct {
		name string
		arr1 []string
		arr2 []string
		want bool
	}{
		{
			name: "empty arrays",
			arr1: []string{},
			arr2: []string{},
			want: true,
		},
		{
			name: "one empty array",
			arr1: []string{},
			arr2: []string{"element"},
			want: false,
		},
		{
			name: "superset",
			arr1: []string{"different element", "element"},
			arr2: []string{"element"},
			want: false,
		},
		{
			name: "duplicates",
			arr1: []string{"different element", "element"},
			arr2: []string{"element", "element", "different element"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			req.Equal(CompareStringArrays(tt.arr1, tt.arr2), tt.want)
		})
	}
}
