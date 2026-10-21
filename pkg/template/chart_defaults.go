package template

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"path"
	"sort"
	"strings"

	semver "github.com/Masterminds/semver/v3"

	"github.com/pkg/errors"

	"go.yaml.in/yaml/v3"
	yamljson "sigs.k8s.io/yaml"
)

// chartDefaults reads only packaged defaults, including dependency aliases.
// It never executes native Helm templates or reads customer-supplied values.
func chartDefaults(archive []byte) (map[string]any, error) {
	files, err := chartDefaultFiles(archive)
	if err != nil {
		return nil, errors.Wrap(err, "read packaged chart defaults")
	}
	return chartDefaultTree(files, 0)
}

func chartDefaultFiles(data []byte) (map[string][]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "open chart gzip archive")
	}
	defer gz.Close()
	files := map[string][]byte{}
	commonDir := ""
	haveFiles := false
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "read chart tar header")
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		name := path.Clean(h.Name)
		if !haveFiles {
			commonDir = path.Dir(name)
			if commonDir == "." {
				commonDir = ""
			}
			haveFiles = true
		}
		for commonDir != "" && !strings.HasPrefix(name, commonDir+"/") {
			commonDir = path.Dir(commonDir)
			if commonDir == "." {
				commonDir = ""
			}
		}
		if path.Base(name) != "Chart.yaml" && path.Base(name) != "values.yaml" && !strings.HasSuffix(name, ".tgz") {
			continue
		}
		files[name], err = io.ReadAll(tr)
		if err != nil {
			return nil, errors.Wrapf(err, "read chart file %s", name)
		}
	}
	// Use the common directory prefix, matching KOTS's existing archive reader.
	prefix := ""
	if commonDir != "" {
		prefix = commonDir + "/"
	}
	normalized := map[string][]byte{}
	for name, data := range files {
		if strings.HasPrefix(name, prefix) {
			normalized[strings.TrimPrefix(name, prefix)] = data
		}
	}
	files = normalized
	if len(files["Chart.yaml"]) == 0 {
		return nil, errors.New("chart archive has no Chart.yaml")
	}
	return files, nil
}

func chartDefaultTree(files map[string][]byte, depth int) (map[string]any, error) {
	if depth > 32 {
		return nil, errors.New("chart dependencies exceed maximum depth")
	}
	var meta struct {
		Name         string
		Dependencies []struct{ Name, Alias, Version string }
	}
	if err := yaml.Unmarshal(files["Chart.yaml"], &meta); err != nil {
		return nil, errors.Wrap(err, "parse Chart.yaml")
	}
	values := map[string]any{}
	if err := yamljson.Unmarshal(files["values.yaml"], &values, func(decoder *json.Decoder) *json.Decoder {
		decoder.UseNumber()
		return decoder
	}); err != nil {
		return nil, errors.Wrapf(err, "parse %s/values.yaml", meta.Name)
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
				return nil, errors.Wrapf(err, "read dependency %s", rest)
			}
			children[rest] = child
		}
	}
	childNames := make([]string, 0, len(children))
	for name := range children {
		childNames = append(childNames, name)
	}
	sort.Strings(childNames)
	assigned := map[string]string{}
	for _, name := range childNames {
		child := children[name]
		var childMeta struct{ Name, Version string }
		if err := yaml.Unmarshal(child["Chart.yaml"], &childMeta); err != nil {
			return nil, errors.Wrapf(err, "parse dependency %s Chart.yaml", name)
		}
		defaults, err := chartDefaultTree(child, depth+1)
		if err != nil {
			return nil, errors.Wrapf(err, "parse dependency %s defaults", name)
		}
		keys := []string{}
		declared := false
		for _, dep := range meta.Dependencies {
			if dep.Name == childMeta.Name {
				declared = true
				if dep.Version != "" {
					constraint, err := semver.NewConstraint(dep.Version)
					if err != nil {
						return nil, errors.Wrapf(err, "parse dependency %s version constraint %q", dep.Name, dep.Version)
					}
					version, err := semver.NewVersion(childMeta.Version)
					if err != nil {
						return nil, errors.Wrapf(err, "parse dependency %s version %q", name, childMeta.Version)
					}
					if !constraint.Check(version) {
						continue
					}
				}
				key := dep.Alias
				if key == "" {
					key = dep.Name
				}
				keys = append(keys, key)
			}
		}
		if len(keys) == 0 {
			if declared {
				continue
			}
			keys = append(keys, childMeta.Name)
		}
		for _, key := range keys {
			if previous, exists := assigned[key]; exists {
				return nil, errors.Errorf("ambiguous dependency %s: both %s and %s supply defaults", key, previous, name)
			}
			assigned[key] = name
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
