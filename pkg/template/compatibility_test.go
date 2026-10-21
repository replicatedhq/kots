package template

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/stretchr/testify/require"
)

func TestSharedFunctionInventory(t *testing.T) {
	data, err := os.ReadFile("testdata/function-contract.json")
	require.NoError(t, err)
	var contract map[string]string
	require.NoError(t, json.Unmarshal(data, &contract))
	b, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)
	builders := []Builder{b}
	for i := range builders {
		functions := builders[i].BuildFuncMap()
		require.Len(t, functions, len(contract))
		for name, signature := range contract {
			require.Contains(t, functions, name)
			actual := reflect.TypeOf(functions[name]).String()
			if name == "Lookup" { // EC propagates live lookup errors; KOTS retains its historical map-only signature.
				require.Contains(t, []string{signature, "func(string, string, string, string) map[string]interface {}"}, actual)
			} else {
				require.Equal(t, signature, actual, name)
			}
		}
		for name := range sprig.TxtFuncMap() {
			require.Contains(t, functions, name)
		}
	}
}

func TestSharedSemanticFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/function-fixtures.json")
	require.NoError(t, err)
	var fixtures []struct{ Expression, Want, Error string }
	require.NoError(t, json.Unmarshal(data, &fixtures))
	b, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)
	for _, fixture := range fixtures {
		t.Run(fixture.Expression, func(t *testing.T) {
			for _, delimiter := range []string{"repl{{ ", "{{repl "} {
				got, err := b.String(delimiter + fixture.Expression + " }}")
				if fixture.Error != "" {
					require.ErrorContains(t, err, fixture.Error)
					continue
				}
				require.NoError(t, err)
				require.Equal(t, fixture.Want, got)
			}
		})
	}
	_, err = b.String(`repl{{ MisspelledFunction }}`)
	require.ErrorContains(t, err, "not defined")
	for _, expression := range []string{`Div 1 0`} {
		_, err := b.String("repl{{ " + expression + " }}")
		require.ErrorContains(t, err, "divide by zero")
	}
	_, err = b.String(`repl{{ KurlInt }}`)
	require.Error(t, err)
}

func TestKOTSFloatingPointDivisionByZero(t *testing.T) {
	b, _, err := NewBuilder(BuilderOptions{})
	require.NoError(t, err)
	tests := []struct{ name, expression, want string }{
		{"positive infinity", "Div 1.0 0.0", "+Inf"},
		{"negative infinity", "Div -1.0 0.0", "-Inf"},
		{"not a number", "Div 0.0 0.0", "NaN"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := b.String("repl{{ " + test.expression + " }}")
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestExistingContextFunctionsKeepPrecedence(t *testing.T) {
	b, _, err := NewBuilder(BuilderOptions{ExistingValues: map[string]ItemValue{"name": {Value: "from-context"}}})
	require.NoError(t, err)
	b.Functs = map[string]interface{}{"ConfigOption": func(string) string { return "from-explicit-map" }}
	got, err := b.String(`repl{{ ConfigOption "name" }}`)
	require.NoError(t, err)
	require.Equal(t, "from-context", got)
}
