package main

import (
	"fmt"
	"testing"

	"github.com/replicatedhq/kots/pkg/image"
	"github.com/stretchr/testify/assert"
)

func Test_ImageNameFromNameParts(t *testing.T) {
	registry := "localhost:5000"

	tests := []struct {
		name          string
		parts         []string
		expectedURL   string
		expectedImage image.Image
		isError       bool
	}{
		{
			name:          "bad name format",
			parts:         []string{"quay.io", "debian", "latest"},
			expectedURL:   "",
			expectedImage: image.Image{},
			isError:       true,
		},
		{
			name:        "four parts with tag",
			parts:       []string{"quay.io", "someorg", "debian", "latest"},
			expectedURL: fmt.Sprintf("%s/someorg/debian:latest", registry),
			expectedImage: image.Image{
				Name:    "quay.io/someorg/debian:latest",
				NewName: fmt.Sprintf("%s/someorg/debian", registry),
				NewTag:  "latest",
				Digest:  ""},
			isError: false,
		},
		{
			name:        "five parts with sha",
			parts:       []string{"quay.io", "someorg", "debian", "sha256", "1234567890abcdef"},
			expectedURL: fmt.Sprintf("%s/someorg/debian@sha256:1234567890abcdef", registry),
			expectedImage: image.Image{
				Name:    "quay.io/someorg/debian@sha256:1234567890abcdef",
				NewName: fmt.Sprintf("%s/someorg/debian", registry),
				NewTag:  "",
				Digest:  "sha256:1234567890abcdef"},
			isError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url, image, err := imageNameFromNameParts(registry, test.parts)
			if test.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedURL, url)
				assert.Equal(t, test.expectedImage, image)
			}
		})
	}
}
