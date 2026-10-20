package template

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/tidwall/gjson"
)

// WithHelmChart returns an independent builder using the packaged chart's defaults.
// Copying the explicit map prevents one chart's context leaking into another.
func (b Builder) WithHelmChart(archive []byte) (Builder, error) {
	defaults, err := chartDefaults(archive)
	if err != nil {
		return Builder{}, fmt.Errorf("load chart defaults: %w", err)
	}
	values, err := json.Marshal(defaults)
	if err != nil {
		return Builder{}, err
	}
	functions := template.FuncMap{}
	for key, fn := range b.Functs {
		functions[key] = fn
	}
	functions["HelmValue"] = func(path string) (string, error) {
		path = strings.TrimPrefix(path, ".Values.")
		path = strings.TrimPrefix(path, ".Values")
		path = strings.TrimPrefix(path, ".")
		return gjson.GetBytes(values, path).String(), nil
	}
	b.Functs = functions
	return b, nil
}
