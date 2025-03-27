package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractReadableText(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty input",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "all readable text",
			input:    []byte("This is all readable text"),
			expected: "This is all readable text",
		},
		{
			name:     "text with newlines and tabs",
			input:    []byte("Line 1\nLine 2\tTabbed"),
			expected: "Line 1\nLine 2\tTabbed",
		},
		{
			name:     "text with binary data at beginning",
			input:    []byte{0x00, 0x01, 0x02, 0x03, 0x04, 'H', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd'},
			expected: "Hello world",
		},
		{
			name:     "text with binary data at end",
			input:    []byte{'H', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd', 0x00, 0x01, 0x02, 0x03, 0x04},
			expected: "Hello world",
		},
		{
			name:     "text with binary data in middle",
			input:    []byte{'S', 't', 'a', 'r', 't', 0x00, 0x01, 0x02, 0x03, 0x04, 'E', 'n', 'd', ' ', ' '},
			expected: "Start ... End  ",
		},
		{
			name:     "multiple text sections separated by binary",
			input:    []byte{'F', 'i', 'r', 's', 't', ' ', 'p', 'a', 'r', 't', 0x00, 0x01, 'S', 'e', 'c', 'o', 'n', 'd', ' ', 'p', 'a', 'r', 't'},
			expected: "First part ... Second part",
		},
		{
			name:     "short text sections (less than 5 chars) are ignored",
			input:    []byte{'A', 'B', 'C', 0x00, 0x01, 0x02, 'D', 'E', 'F', 'G', 'H', 0x03, 'I', 'J'},
			expected: "DEFGH",
		},
		{
			name:     "binary data with no readable text",
			input:    []byte{0x00, 0x01, 0x02, 0x03, 0x04},
			expected: "",
		},
		{
			name:     "real world example: gzip header with text",
			input:    []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 'E', 'r', 'r', 'o', 'r', ':', ' ', 'i', 'n', 'v', 'a', 'l', 'i', 'd', ' ', 'g', 'z', 'i', 'p', ' ', 'h', 'e', 'a', 'd', 'e', 'r'},
			expected: "Error: invalid gzip header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractReadableText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
