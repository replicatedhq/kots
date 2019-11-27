package template

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

type testContext struct {
}

func (ctx testContext) FuncMap() template.FuncMap {
	return map[string]interface{}{
		"ConfigOption": func(key string) interface{} {
			switch key {
			case "option_1":
				return "Option 1"
			case "option_2":
				return "Option 2"
			}
			return ""
		},
	}
}

type testcase interface {
	name() string
	runTest(t *testing.T, builder Builder)
}

type strcase struct {
	Name     string
	Template string
	Expected string
}

func (c strcase) name() string {
	return c.Name
}
func (c strcase) runTest(t *testing.T, builder Builder) {
	built, _ := builder.String(c.Template)
	require.New(t).Equal(c.Expected, built)
}

type intcase struct {
	Name     string
	Template string
	Expected int64
}

func (c intcase) name() string {
	return c.Name
}
func (c intcase) runTest(t *testing.T, builder Builder) {
	built, _ := builder.Int(c.Template, 0)
	require.New(t).Equal(c.Expected, built)
}

type uintcase struct {
	Name     string
	Template string
	Expected uint64
}

func (c uintcase) name() string {
	return c.Name
}
func (c uintcase) runTest(t *testing.T, builder Builder) {
	built, _ := builder.Uint(c.Template, 0)
	require.New(t).Equal(c.Expected, built)
}

type floatcase struct {
	Name     string
	Template string
	Expected float64
}

func (c floatcase) name() string {
	return c.Name
}
func (c floatcase) runTest(t *testing.T, builder Builder) {
	built, _ := builder.Float64(c.Template, 0.0)
	require.New(t).Equal(c.Expected, built)
}

type boolcase struct {
	Name     string
	Template string
	Expected bool
}

func (c boolcase) name() string {
	return c.Name
}
func (c boolcase) runTest(t *testing.T, builder Builder) {
	built, _ := builder.Bool(c.Template, false)
	require.New(t).Equal(c.Expected, built)
}

func TestBuildStrings(t *testing.T) {
	// "Now"
	// "NowFmt"
	cases := []testcase{
		strcase{
			Name:     "Test ToLower",
			Template: `{{repl ToLower "ABcd87-()"}}`,
			Expected: "abcd87-()",
		},
		strcase{
			Name:     "Test ToUpper",
			Template: `repl{{ (ToUpper "ABcd87-()") }}`,
			Expected: "ABCD87-()",
		},
		strcase{
			Name:     "Test TrimSpace",
			Template: `{{repl TrimSpace "   maybe spaces,\tmaybe not \t  "}}`,
			Expected: "maybe spaces,\tmaybe not",
		},
		strcase{
			Name:     "Test Trim",
			Template: `{{repl Trim "aaaaamaybe spaces,\tmaybe not \taaa" "a"}}`,
			Expected: "maybe spaces,\tmaybe not \t",
		},
		strcase{
			Name:     "Test UrlEncode",
			Template: `repl{{ UrlEncode "?some url unsafe=%value"}}`,
			Expected: "%3Fsome+url+unsafe%3D%25value",
		},
		strcase{
			Name:     "Test Base64Encode",
			Template: `{{repl Base64Encode "clear text"}}`,
			Expected: "Y2xlYXIgdGV4dA==",
		},
		strcase{
			Name:     "Test Base64Decode",
			Template: `{{repl Base64Decode "Y2xlYXIgdGV4dA=="}}`,
			Expected: "clear text",
		},
		strcase{
			Name:     "Test Split",
			Template: `{{repl Split "1,2,3,4" ","}}`,
			Expected: "[1 2 3 4]",
		},
		strcase{
			Name:     "Test HumanSize",
			Template: `{{repl HumanSize 387346587344}}`,
			Expected: "387.3GB",
		},
		strcase{
			Name:     "Test ConfigOption",
			Template: `{{repl ConfigOption "option_1"}}`,
			Expected: "Option 1",
		},
		strcase{
			Name:     "Test ConfigOption",
			Template: `{{repl ConfigOption "option_2"}}`,
			Expected: "Option 2",
		},
		strcase{
			Name:     "Test ConfigOption",
			Template: `{{repl ConfigOption "option_3"}}`,
			Expected: "",
		},
		intcase{
			Name:     "Test Mult",
			Template: `{{repl Mult 34 2}}`,
			Expected: 68,
		},
		intcase{
			Name:     "Test ParseInt",
			Template: `{{repl ParseInt "-15"}}`,
			Expected: -15,
		},
		uintcase{
			Name:     "Test Add",
			Template: `{{repl Add 2 3}}`,
			Expected: 5,
		},
		uintcase{
			Name:     "Test ParseUint",
			Template: `repl{{ ParseUint "15"}}`,
			Expected: 15,
		},
		floatcase{
			Name:     "Test Sub",
			Template: `{{repl Sub 1.5 1}}`,
			Expected: 0.5,
		},
		floatcase{
			Name:     "Test Div",
			Template: `{{repl Div 15.0 2}}`,
			Expected: 7.5,
		},
		floatcase{
			Name:     "Test ParseFloat",
			Template: `{{repl ParseFloat "1.5"}}`,
			Expected: 1.5,
		},
		boolcase{
			Name:     "Test ParseBool",
			Template: `{{repl ParseBool "True"}}`,
			Expected: true,
		},
	}

	builder := Builder{}
	builder.AddCtx(StaticCtx{})
	builder.AddCtx(testContext{})

	for _, test := range cases {
		t.Run(test.name(), func(t *testing.T) {
			test.runTest(t, builder)
		})
	}

	t.Run("Test bad values", func(t *testing.T) {
		built, err := builder.String("{{repl ParseBool True}}")
		require.New(t).Error(err)
		require.New(t).Equal("", built)
	})

	t.Run("Test broken template syntax", func(t *testing.T) {
		built, err := builder.String("{{repl SomeFunc")
		require.New(t).Error(err)
		require.New(t).Equal("", built)
	})
}
