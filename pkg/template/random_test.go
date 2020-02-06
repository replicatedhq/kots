package template

import (
	"fmt"
	_ "regexp"
	"testing"
	_ "unicode/utf8"

	"github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
)

func TestGenerateRandomString(t *testing.T) {
	ctx := &StaticCtx{}
	str := ctx.RandomString(100)
	assert.Len(t, str, 100)
	assert.Regexp(t, DefaultCharset, str)
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
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()
			// TODO

			// req := require.New(t)

			// builderBuilder := &BuilderBuilder{
			// 	Logger: &logger.TestLogger{T: t},
			// 	Viper:  viper.New(),
			// }

			// builder := builderBuilder.NewBuilder(
			// 	builderBuilder.NewStaticContext(),
			// )

			// outputString, err := builder.String(tt.templateString)
			// req.NoError(err)

			// req.Equal(utf8.RuneCountInString(outputString), tt.length, "%q should have length %d", outputString, tt.length)

			// tt.outputRegex = fmt.Sprintf("^%s+$", tt.outputRegex)

			// matcher := regexp.MustCompile(tt.outputRegex)

			// req.True(matcher.MatchString(outputString), "String %q must match %q", outputString, tt.outputRegex)
		})
	}
}
