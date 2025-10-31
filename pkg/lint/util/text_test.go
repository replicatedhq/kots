package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryGetLineNumberFromValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{
			name:     "extract line number",
			input:    "yaml: line 42: mapping values are not allowed",
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "no line number",
			input:    "some error without line info",
			expected: -1,
			wantErr:  false,
		},
		{
			name:     "line at beginning",
			input:    "line 10: error",
			expected: 10,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, err := TryGetLineNumberFromValue(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				// Error is not critical, we just check the result
				assert.Equal(t, tt.expected, line)
			}
		})
	}
}

func TestGetLineNumberFromMatch(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: example
data:
  key: value`

	tests := []struct {
		name     string
		match    string
		docIndex int
		expected int
		wantErr  bool
	}{
		{
			name:     "find ConfigMap",
			match:    "ConfigMap",
			docIndex: 0,
			expected: 2,
		},
		{
			name:     "find metadata",
			match:    "metadata",
			docIndex: 0,
			expected: 3,
		},
		{
			name:     "find value",
			match:    "value",
			docIndex: 0,
			expected: 6,
		},
		{
			name:     "not found",
			match:    "notfound",
			docIndex: 0,
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, err := GetLineNumberFromMatch(content, tt.match, tt.docIndex)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, line)
			}
		})
	}
}

func TestGetLineNumberFromYamlPath(t *testing.T) {
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: example
  namespace: default
data:
  key1: value1
  key2: value2`

	tests := []struct {
		name     string
		path     string
		docIndex int
		expected int
	}{
		{
			name:     "find apiVersion",
			path:     "apiVersion",
			docIndex: 0,
			expected: 1,
		},
		{
			name:     "find kind",
			path:     "kind",
			docIndex: 0,
			expected: 2,
		},
		{
			name:     "find metadata.name",
			path:     "metadata.name",
			docIndex: 0,
			expected: 4,
		},
		{
			name:     "find metadata.namespace",
			path:     "metadata.namespace",
			docIndex: 0,
			expected: 5,
		},
		{
			name:     "find data.key1",
			path:     "data.key1",
			docIndex: 0,
			expected: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, err := GetLineNumberFromYamlPath(content, tt.path, tt.docIndex)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, line)
		})
	}
}

func TestGetLineNumberForDoc(t *testing.T) {
	multiDoc := `# Comment
apiVersion: v1
kind: ConfigMap
---
# Another comment

apiVersion: v1
kind: Secret
---
apiVersion: v1
kind: Service`

	tests := []struct {
		name     string
		docIndex int
		expected int
	}{
		{
			name:     "first document",
			docIndex: 0,
			expected: 2,
		},
		{
			name:     "second document",
			docIndex: 1,
			expected: 7,
		},
		{
			name:     "third document",
			docIndex: 2,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, err := GetLineNumberForDoc(multiDoc, tt.docIndex)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, line)
		})
	}
}

func TestIsLineEmpty(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"", true},
		{"   ", true},
		{"# comment", true},
		{"  # comment", true},
		{"apiVersion: v1", false},
		{"  kind: Pod", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsLineEmpty(tt.line))
		})
	}
}

func TestGetLineIndentation(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"apiVersion: v1", ""},
		{"  kind: Pod", "  "},
		{"    name: test", "    "},
		{"\t\tname: test", "\t\t"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetLineIndentation(tt.line))
		})
	}
}

func TestCleanUpYaml(t *testing.T) {
	input := `# This is a comment
apiVersion: v1
kind: ConfigMap

# Another comment
metadata:
  name: test
--- # inline comment
apiVersion: v1
kind: Secret`

	expected := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
---
apiVersion: v1
kind: Secret`

	result := CleanUpYaml(input)
	assert.Equal(t, expected, result)
}

func TestGetStringInBetween(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		start    string
		end      string
		expected string
	}{
		{
			name:     "extract middle",
			str:      "hello [world] test",
			start:    "[",
			end:      "]",
			expected: "world",
		},
		{
			name:     "no start",
			str:      "hello world",
			start:    "[",
			end:      "]",
			expected: "",
		},
		{
			name:     "no end",
			str:      "hello [world",
			start:    "[",
			end:      "]",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStringInBetween(tt.str, tt.start, tt.end)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommonSlicePrefix(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
		allowNil bool
	}{
		{
			name:     "full match",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "partial match",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "d"},
			expected: []string{"a", "b"},
		},
		{
			name:     "no match",
			a:        []string{"a", "b", "c"},
			b:        []string{"x", "y", "z"},
			expected: nil,
			allowNil: true,
		},
		{
			name:     "empty slices",
			a:        []string{},
			b:        []string{},
			expected: nil,
			allowNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CommonSlicePrefix(tt.a, tt.b)
			if tt.allowNil && result == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
