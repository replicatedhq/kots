package main

import (
	"testing"
	"fmt"

	"github.com/stretchr/testify/assert"
)

func Test_ImageNameFromNameParts(t *testing.T) {
	registry := "localhost:5000"

	tests := []struct {
		name string
		parts []string
		expected    string
		isError bool
	}{
		{
			name: "bad name format",
			parts:  []string{"quay.io", "debian", "latest"},
			expected: "",
			isError: true,
		},
		{
			name: "four parts with tag",
			parts:  []string{"quay.io", "someorg", "debian", "latest"},
			expected: fmt.Sprintf("%s/someorg/debian:latest", registry),
			isError: false,
		},
		{
			name: "five parts with sha",
			parts:  []string{"quay.io", "someorg", "debian", "sha256", "1234567890abcdef"},
			expected: fmt.Sprintf("%s/someorg/debian@sha256:1234567890abcdef", registry),
			isError: false,
		},
	}


	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url, err := imageNameFromNameParts(registry, test.parts)
			if test.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, url)
			}
		})
	}
}
