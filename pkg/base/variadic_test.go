package base

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_generateTargetValue(t *testing.T) {
	tests := []struct {
		name             string
		configOptionName string
		valueName        string
		target           string
		templateValue    interface{}
		want             interface{}
	}{
		{
			configOptionName: "secret",
			valueName:        "secret-1",
			target:           "repl{{ ConfigOption \"secret\" }}",
			templateValue:    "123",
			want:             "repl{{ ConfigOption \"secret-1\" }}",
		},
		{
			configOptionName: "secret",
			valueName:        "secret-1",
			target:           "repl{{ ConfigOptionName \"secret\" }}",
			templateValue:    "123",
			want:             "repl{{ ConfigOptionName \"secret-1\" }}",
		},
		{
			configOptionName: "secret",
			valueName:        "secret-1",
			target:           "repl{{ ConfigOptionFilename \"secret\" }}",
			templateValue:    "123",
			want:             "repl{{ ConfigOptionFilename \"secret-1\" }}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)
			result := generateTargetValue(test.configOptionName, test.valueName, test.target, test.templateValue)

			req.Equal(test.want, result)
		})
	}
}
