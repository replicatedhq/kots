package multitype

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestQuotedBool_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  QuotedBool
	}{
		{
			name:  "actual string",
			input: `"hello world"`,
			want:  QuotedBool("hello world"),
		},
		{
			name:  "quoted boolstring",
			input: `"true"`,
			want:  QuotedBool("true"),
		},
		{
			name:  "unquoted boolstring",
			input: `true`,
			want:  QuotedBool("true"),
		},
		{
			name:  "unquoted boolstring false",
			input: `false`,
			want:  QuotedBool("false"),
		},
		{
			name:  "unquoted boolstring 0",
			input: `0`,
			want:  QuotedBool("false"),
		},
		{
			name:  "unquoted boolstring 1",
			input: `1`,
			want:  QuotedBool("true"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			var newQuotBool QuotedBool

			err := json.Unmarshal([]byte(tt.input), &newQuotBool)
			req.NoError(err)
			req.Equal(tt.want, newQuotBool)
		})
	}
}

func TestQuotedBool_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  QuotedBool
	}{
		{
			name:  "actual string",
			input: `"hello world"`,
			want:  QuotedBool("hello world"),
		},
		{
			name:  "quoted boolstring",
			input: `"true"`,
			want:  QuotedBool("true"),
		},
		{
			name:  "unquoted boolstring",
			input: `true`,
			want:  QuotedBool("true"),
		},
		{
			name:  "unquoted boolstring false",
			input: `false`,
			want:  QuotedBool("false"),
		},
		{
			name:  "unquoted string",
			input: `hello world`,
			want:  QuotedBool("hello world"),
		},
		{
			name:  "unquoted boolstring no",
			input: `no`,
			want:  QuotedBool("false"),
		},
		{
			name:  "unquoted boolstring 0",
			input: `0`,
			want:  QuotedBool("false"),
		},
		{
			name:  "unquoted boolstring 1",
			input: `1`,
			want:  QuotedBool("true"),
		},
		{
			name:  "unquoted boolstring 45", // not a valid input, but we'll count it as true
			input: `45`,
			want:  QuotedBool("true"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			var newQuotBool QuotedBool

			err := yaml.Unmarshal([]byte(tt.input), &newQuotBool)
			req.NoError(err)
			req.Equal(tt.want, newQuotBool)
		})
	}
}
