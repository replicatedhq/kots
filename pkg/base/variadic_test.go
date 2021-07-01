package base

import "testing"

func Test_generateTargetValue(t *testing.T) {
	tests := []struct {
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
			want:             "123",
		},
		{
			configOptionName: "secret",
			valueName:        "secret-1",
			target:           "repl{{ RepeatOptionName \"secret\" }}",
			templateValue:    "123",
			want:             "secret-1",
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
		result := generateTargetValue(test.configOptionName, test.valueName, test.target, test.templateValue)
		if result != test.want {
			t.Errorf("generateTargetValue() failed: want: %v\ngot: %v", test.want, result)
		}
	}
}
