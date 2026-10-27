package template

import (
	"encoding/json"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/tidwall/gjson"
)

// WithHelmChart returns an independent builder using the packaged chart's defaults.
// Copying the explicit map prevents one chart's context leaking into another.
func (b Builder) WithHelmChart(archive []byte) (Builder, error) {
	defaults, err := chartDefaults(archive)
	if err != nil {
		return Builder{}, errors.Wrap(err, "load chart defaults")
	}
	values, err := json.Marshal(defaults)
	if err != nil {
		return Builder{}, errors.Wrap(err, "encode chart defaults")
	}
	functions := template.FuncMap{}
	for key, fn := range b.Functs {
		functions[key] = fn
	}
	functions["HelmValue"] = func(path string) (string, error) {
		switch {
		case path == ".Values":
			path = ""
		case strings.HasPrefix(path, ".Values."):
			path = strings.TrimPrefix(path, ".Values.")
		default:
			path = strings.TrimPrefix(path, ".")
		}
		return gjson.GetBytes(values, path).String(), nil
	}
	b.Functs = functions
	return b, nil
}

// helmChartCtx supplies the empty HelmValue default outside a matching chart.
type helmChartCtx struct{}

func (helmChartCtx) FuncMap() template.FuncMap {
	return template.FuncMap{"HelmValue": func(string) (string, error) { return "", nil }}
}
