package template

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestGenerateRandomString(t *testing.T) {
	req := require.New(t)
	ctx := &StaticCtx{}
	seenStrings := map[string]struct{}{}
	for i := 0; i < 100; i++ {
		str := ctx.RandomString(100)
		req.Len(str, 100)
		req.Regexp(DefaultCharset, str)
		_, ok := seenStrings[str]
		req.Falsef(ok, "string %q matched an earlier random string of length 100 on iteration %d", str, i)
		seenStrings[str] = struct{}{}
	}
}

func TestGenerateRandomStringTemplates(t *testing.T) {
	tests := []struct {
		name           string
		length         int
		outputRegex    string
		templateString string
	}{
		{
			name:           "DefaultCharset",
			length:         100,
			outputRegex:    DefaultCharset,
			templateString: `{{repl RandomString 100}}`,
		},
		{
			name:           "ExplicitDefaultCharset",
			length:         100,
			outputRegex:    DefaultCharset,
			templateString: fmt.Sprintf("{{repl RandomString 100 %q}}", DefaultCharset),
		},
		{
			name:           "lowercase",
			length:         100,
			outputRegex:    "[a-z]",
			templateString: `{{repl RandomString 100 "[a-z]"}}`,
		},
		{
			name:           "very restricted charset aB",
			length:         100,
			outputRegex:    "[aB]",
			templateString: `{{repl RandomString 100 "[aB]"}}`,
		},
		{
			name:           "not lowercase",
			length:         1000,
			outputRegex:    "[^a-z]",
			templateString: `{{repl RandomString 1000 "[^a-z]"}}`,
		},
		{
			name:           "base64",
			length:         1000,
			outputRegex:    "[A-Za-z0-9+/]",
			templateString: `{{repl RandomString 1000 "[A-Za-z0-9+/]"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			builder := Builder{}
			builder.AddCtx(StaticCtx{})

			seenStrings := map[string]struct{}{}
			for i := 0; i < 100; i++ {
				outputString, err := builder.String(tt.templateString)
				req.NoError(err)

				req.Equal(utf8.RuneCountInString(outputString), tt.length, "%q should have length %d", outputString, tt.length)

				matcher := regexp.MustCompile(fmt.Sprintf("^%s+$", tt.outputRegex))
				req.Regexp(matcher, outputString)

				_, ok := seenStrings[outputString]
				req.Falsef(ok, "string %q matched an earlier random string of length 100 on iteration %d", outputString, i)
				seenStrings[outputString] = struct{}{}
			}
		})
	}
}

func TestRandomBytes(t *testing.T) {
	req := require.New(t)
	ctx := &StaticCtx{}
	seenStrings := map[string]struct{}{}
	for i := 0; i < 100; i++ {
		str := ctx.RandomBytes(100)
		bytes, err := base64.StdEncoding.DecodeString(str)
		req.NoError(err)
		req.Len(bytes, 100)

		_, ok := seenStrings[str]
		req.Falsef(ok, "string %q matched an earlier random string of length 100 on iteration %d", str, i)
		seenStrings[str] = struct{}{}
	}
}
