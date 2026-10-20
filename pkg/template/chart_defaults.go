package template

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path"
	"strings"

	"go.yaml.in/yaml/v3"
)

// chartDefaults reads only packaged defaults, including dependency aliases.
// It never executes native Helm templates or reads customer-supplied values.
func chartDefaults(archive []byte) (map[string]any, error) {
	files, err := chartDefaultFiles(archive)
	if err != nil {
		return nil, err
	}
	return chartDefaultTree(files, 0)
}

func chartDefaultFiles(data []byte) (map[string][]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	files := map[string][]byte{}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		_, name, ok := strings.Cut(h.Name, "/")
		if !ok {
			continue
		}
		if path.Base(name) != "Chart.yaml" && path.Base(name) != "values.yaml" && !(strings.HasPrefix(name, "charts/") && strings.HasSuffix(name, ".tgz")) {
			continue
		}
		files[name], err = io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
	}
	if len(files["Chart.yaml"]) == 0 {
		return nil, fmt.Errorf("chart archive has no Chart.yaml")
	}
	return files, nil
}

func chartDefaultTree(files map[string][]byte, depth int) (map[string]any, error) {
	if depth > 32 {
		return nil, fmt.Errorf("chart dependencies exceed maximum depth")
	}
	var meta struct {
		Name         string
		Dependencies []struct{ Name, Alias string }
	}
	if err := yaml.Unmarshal(files["Chart.yaml"], &meta); err != nil {
		return nil, fmt.Errorf("parse Chart.yaml: %w", err)
	}
	values := map[string]any{}
	if err := yaml.Unmarshal(files["values.yaml"], &values); err != nil {
		return nil, fmt.Errorf("parse %s/values.yaml: %w", meta.Name, err)
	}
	if values == nil {
		values = map[string]any{}
	}
	children := map[string]map[string][]byte{}
	for name, data := range files {
		rest, ok := strings.CutPrefix(name, "charts/")
		if !ok {
			continue
		}
		if dir, file, ok := strings.Cut(rest, "/"); ok {
			if children[dir] == nil {
				children[dir] = map[string][]byte{}
			}
			children[dir][file] = data
		} else if strings.HasSuffix(rest, ".tgz") {
			child, err := chartDefaultFiles(data)
			if err != nil {
				return nil, fmt.Errorf("read dependency %s: %w", rest, err)
			}
			children[rest] = child
		}
	}
	for name, child := range children {
		var childMeta struct{ Name string }
		if err := yaml.Unmarshal(child["Chart.yaml"], &childMeta); err != nil {
			return nil, fmt.Errorf("parse dependency %s Chart.yaml: %w", name, err)
		}
		defaults, err := chartDefaultTree(child, depth+1)
		if err != nil {
			return nil, err
		}
		keys := []string{}
		for _, dep := range meta.Dependencies {
			if dep.Name == childMeta.Name {
				key := dep.Alias
				if key == "" {
					key = dep.Name
				}
				keys = append(keys, key)
			}
		}
		if len(keys) == 0 {
			keys = append(keys, childMeta.Name)
		}
		for _, key := range keys {
			overrides, present := values[key]
			if present && overrides == nil {
				continue
			} // Helm null removes a dependency value.
			if overrides, ok := overrides.(map[string]any); ok {
				values[key] = mergeChartDefaults(defaults, overrides)
			} else if !present {
				values[key] = defaults
			}
		}
	}
	return values, nil
}

func mergeChartDefaults(base, overrides map[string]any) map[string]any {
	result := map[string]any{}
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overrides {
		if v == nil {
			delete(result, k)
			continue
		}
		child, ok := v.(map[string]any)
		previous, previousOK := result[k].(map[string]any)
		if ok && previousOK {
			result[k] = mergeChartDefaults(previous, child)
		} else {
			result[k] = v
		}
	}
	return result
}
