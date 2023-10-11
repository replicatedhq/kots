package upstream

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"gopkg.in/yaml.v3"
)

// configureChart will configure the chart archive (values.yaml),
// repackage it, and return the updated content of the chart
func configureChart(chartContent []byte, u *types.Upstream, options types.WriteOptions) ([]byte, error) {
	replicatedChartName, isSubchart, err := findReplicatedChart(bytes.NewReader(chartContent), u.ReplicatedChartNames)
	if err != nil {
		return nil, errors.Wrap(err, "find replicated chart")
	}
	if replicatedChartName == "" {
		return chartContent, nil
	}

	chartValues, pathInArchive, extractedArchiveRoot, err := findTopLevelChartValues(bytes.NewReader(chartContent))
	if err != nil {
		return nil, errors.Wrap(err, "find top level chart values")
	}
	defer os.RemoveAll(extractedArchiveRoot)

	updatedValues, err := configureChartValues(chartValues, replicatedChartName, isSubchart, u, options)
	if err != nil {
		return nil, errors.Wrap(err, "configure values yaml")
	}

	if err := os.WriteFile(filepath.Join(extractedArchiveRoot, pathInArchive), updatedValues, 0644); err != nil {
		return nil, errors.Wrap(err, "write configured values.yaml")
	}

	updatedArchive, err := packageChartArchive(extractedArchiveRoot)
	if err != nil {
		return nil, errors.Wrap(err, "package chart archive")
	}
	defer os.RemoveAll(updatedArchive)

	updatedContents, err := os.ReadFile(updatedArchive)
	if err != nil {
		return nil, errors.Wrap(err, "read updated archive")
	}

	return updatedContents, nil
}

// findReplicatedChart will look for the replicated chart in the archive
// and return the name of the replicated chart and whether it is the parent chart or a subchart
func findReplicatedChart(chartArchive io.Reader, replicatedChartNames []string) (string, bool, error) {
	gzReader, err := gzip.NewReader(chartArchive)
	if err != nil {
		return "", false, errors.Wrap(err, "failed to create gzip reader")
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false, errors.Wrap(err, "failed to read header from tar")
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if filepath.Base(header.Name) != "Chart.yaml" {
				continue
			}

			// we only care about the root Chart.yaml file or the Chart.yaml file of direct subcharts (not subsubcharts)
			parts := strings.Split(header.Name, string(os.PathSeparator)) // e.g. replicated/Chart.yaml or nginx/charts/replicated/Chart.yaml
			if len(parts) != 2 && len(parts) != 4 {
				continue
			}

			content, err := io.ReadAll(tarReader)
			if err != nil {
				return "", false, errors.Wrapf(err, "failed to read file %s", header.Name)
			}

			chartInfo := struct {
				ChartName string `json:"name" yaml:"name"`
			}{}
			if err := yaml.Unmarshal(content, &chartInfo); err != nil {
				return "", false, errors.Wrapf(err, "failed to unmarshal %s", header.Name)
			}

			for _, replicatedChartName := range replicatedChartNames {
				if chartInfo.ChartName == replicatedChartName {
					return replicatedChartName, len(parts) == 4, nil
				}
			}
		}
	}

	return "", false, nil
}

func findTopLevelChartValues(r io.Reader) (valuesYaml []byte, pathInArchive string, workspace string, finalErr error) {
	workspace, err := os.MkdirTemp("", "extracted-chart-")
	if err != nil {
		finalErr = errors.Wrap(err, "failed to create temp directory")
		return
	}

	defer func() {
		if finalErr != nil {
			os.RemoveAll(workspace)
			workspace = ""
		}
	}()

	gzReader, err := gzip.NewReader(r)
	if err != nil {
		finalErr = errors.Wrap(err, "failed to create gzip reader")
		return
	}

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			finalErr = errors.Wrap(err, "failed to read header from tar")
			return
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(filepath.Join(workspace, header.Name), fs.FileMode(header.Mode)); err != nil {
				finalErr = errors.Wrap(err, "failed to create directory from archive")
				return
			}
		case tar.TypeReg:
			content, err := io.ReadAll(tarReader)
			if err != nil {
				finalErr = errors.Wrap(err, "failed to read file")
				return
			}

			if filepath.Base(header.Name) == "values.yaml" {
				// only get the values.yaml from the top level chart
				p := filepath.Dir(header.Name)
				if !strings.Contains(p, string(os.PathSeparator)) {
					pathInArchive = header.Name
					valuesYaml = content
				}
			}

			dir := filepath.Dir(filepath.Join(workspace, header.Name))
			if err := os.MkdirAll(dir, 0700); err != nil {
				finalErr = errors.Wrap(err, "failed to create directory from filename")
				return
			}

			outFile, err := os.Create(filepath.Join(workspace, header.Name))
			if err != nil {
				finalErr = errors.Wrap(err, "failed to create file")
				return
			}
			defer outFile.Close()
			if err := os.WriteFile(outFile.Name(), content, header.FileInfo().Mode()); err != nil {
				finalErr = errors.Wrap(err, "failed to write file")
				return
			}
		}
	}

	return
}

func configureChartValues(valuesYAML []byte, replicatedChartName string, isSubchart bool, u *types.Upstream, options types.WriteOptions) ([]byte, error) {
	// unmarshal to insert the replicated values
	var valuesNode yaml.Node
	if err := yaml.Unmarshal([]byte(valuesYAML), &valuesNode); err != nil {
		return nil, errors.Wrap(err, "unmarshal values")
	}

	if len(valuesNode.Content) == 0 {
		return nil, errors.New("no content")
	}

	if replicatedChartName != "" {
		err := addReplicatedValues(valuesNode.Content[0], replicatedChartName, isSubchart, u, options)
		if err != nil {
			return nil, errors.Wrap(err, "add replicated values")
		}
	}

	if err := addGlobalReplicatedValues(valuesNode.Content[0], u, options); err != nil {
		return nil, errors.Wrap(err, "add global replicated values")
	}

	updatedValues, err := kotsutil.NodeToYAML(&valuesNode)
	if err != nil {
		return nil, errors.Wrap(err, "node to yaml")
	}

	return updatedValues, nil
}

func addReplicatedValues(doc *yaml.Node, replicatedChartName string, isSubchart bool, u *types.Upstream, options types.WriteOptions) error {
	replicatedValues, err := buildReplicatedValues(u, options)
	if err != nil {
		return errors.Wrap(err, "build replicated values")
	}

	targetNode := doc
	hasReplicatedValues := false
	v := replicatedValues

	// if replicated sdk is included as a subchart,
	// we make sure to add the values under the subchart name
	// as helm expects the field name to match the subchart name
	if isSubchart {
		for i, n := range doc.Content {
			if n.Value == replicatedChartName { // check if field already exists
				targetNode = doc.Content[i+1]
				hasReplicatedValues = true
				break
			}
		}
		if !hasReplicatedValues {
			v = map[string]interface{}{
				replicatedChartName: replicatedValues,
			}
		}
	}

	additionalYAML, err := yaml.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "marshal additional values")
	}

	var additionalNode yaml.Node
	if err := yaml.Unmarshal([]byte(additionalYAML), &additionalNode); err != nil {
		return errors.Wrap(err, "unmarshal additional values")
	}

	if !hasReplicatedValues && isSubchart {
		targetNode.Content = append(targetNode.Content, additionalNode.Content[0].Content...)
	} else {
		targetNode.Content = kotsutil.MergeYAMLNodes(targetNode.Content, additionalNode.Content[0].Content)
	}

	return nil
}

func buildReplicatedValues(u *types.Upstream, options types.WriteOptions) (map[string]interface{}, error) {
	replicatedValues := map[string]interface{}{
		"replicatedID": options.KotsadmID,
		"appID":        options.AppID,
		"userAgent":    buildversion.GetUserAgent(),
		"isAirgap":     options.IsAirgap,
	}

	// only add the license if this is an airgap install
	// because the airgap builder doesn't have the license context
	if u.License != nil && options.IsAirgap {
		replicatedValues["license"] = string(MustMarshalLicense(u.License))
	}

	return replicatedValues, nil
}

func addGlobalReplicatedValues(doc *yaml.Node, u *types.Upstream, options types.WriteOptions) error {
	globalReplicatedValues, err := buildGlobalReplicatedValues(u, options)
	if err != nil {
		return errors.Wrap(err, "build global replicated values")
	}
	if len(globalReplicatedValues) == 0 {
		return nil
	}

	targetNode := doc
	hasGlobal := false
	for i, n := range doc.Content {
		if n.Value == "global" {
			targetNode = doc.Content[i+1]
			hasGlobal = true
			break
		}
	}

	hasGlobalReplicated := false
	if hasGlobal {
		for i, n := range targetNode.Content {
			if n.Value == "replicated" {
				targetNode = targetNode.Content[i+1]
				hasGlobalReplicated = true
				break
			}
		}
	}

	v := globalReplicatedValues
	if !hasGlobalReplicated {
		v = map[string]interface{}{
			"replicated": v,
		}
		if !hasGlobal {
			v = map[string]interface{}{
				"global": v,
			}
		}
	}

	additionalYAML, err := yaml.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "marshal additional values")
	}

	var additionalNode yaml.Node
	if err := yaml.Unmarshal([]byte(additionalYAML), &additionalNode); err != nil {
		return errors.Wrap(err, "unmarshal additional values")
	}

	if hasGlobalReplicated || hasGlobal {
		targetNode.Content = kotsutil.MergeYAMLNodes(targetNode.Content, additionalNode.Content[0].Content)
	} else {
		targetNode.Content = append(targetNode.Content, additionalNode.Content[0].Content...)
	}

	return nil
}

func buildGlobalReplicatedValues(u *types.Upstream, options types.WriteOptions) (map[string]interface{}, error) {
	globalReplicatedValues := map[string]interface{}{}

	// only add license related info if this is an airgap install
	// because the airgap builder doesn't have the license context
	if u.License != nil && options.IsAirgap {
		globalReplicatedValues["channelName"] = u.License.Spec.ChannelName
		globalReplicatedValues["customerName"] = u.License.Spec.CustomerName
		globalReplicatedValues["customerEmail"] = u.License.Spec.CustomerEmail
		globalReplicatedValues["licenseID"] = u.License.Spec.LicenseID
		globalReplicatedValues["licenseType"] = u.License.Spec.LicenseType

		// we marshal and then unmarshal entitlements into an interface to evaluate entitlement values
		// and end up with a single value instead of (intVal, boolVal, strVal, and type)
		marshalledEntitlements, err := json.Marshal(u.License.Spec.Entitlements)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal entitlements")
		}

		var licenseFields map[string]interface{}
		if err := json.Unmarshal(marshalledEntitlements, &licenseFields); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal entitlements")
		}

		// add the field name if missing
		for k, v := range licenseFields {
			if name, ok := v.(map[string]interface{})["name"]; !ok || name == "" {
				licenseFields[k].(map[string]interface{})["name"] = k
			}
		}

		globalReplicatedValues["licenseFields"] = licenseFields

		// add docker config json
		auth := fmt.Sprintf("%s:%s", u.License.Spec.LicenseID, u.License.Spec.LicenseID)
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		dockercfg := map[string]interface{}{
			"auths": map[string]interface{}{
				u.ReplicatedProxyDomain: map[string]string{
					"auth": encodedAuth,
				},
				u.ReplicatedRegistryDomain: map[string]string{
					"auth": encodedAuth,
				},
			},
		}

		b, err := json.Marshal(dockercfg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal dockercfg")
		}

		globalReplicatedValues["dockerconfigjson"] = base64.StdEncoding.EncodeToString(b)
	}

	return globalReplicatedValues, nil
}

func packageChartArchive(extractedArchiveRoot string) (string, error) {
	configuredChartArchive, err := os.CreateTemp("", "configured-chart-")
	if err != nil {
		return "", errors.Wrap(err, "create temp file")
	}

	gzipWriter := gzip.NewWriter(configuredChartArchive)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	err = filepath.Walk(extractedArchiveRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return errors.Wrapf(err, "open file '%s'", path)
		}
		defer file.Close()

		rel, err := filepath.Rel(extractedArchiveRoot, path)
		if err != nil {
			return errors.New(fmt.Sprintf("Could not get relative path for file '%s', got error '%s'", path, err.Error()))
		}

		header := &tar.Header{
			Name:    rel,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}

		err = tarWriter.WriteHeader(header)
		if err != nil {
			return errors.New(fmt.Sprintf("Could not write header for file '%s', got error '%s'", path, err.Error()))
		}

		_, err = io.Copy(tarWriter, file)
		if err != nil {
			return errors.New(fmt.Sprintf("Could not copy the file '%s' data to the tarball, got error '%s'", path, err.Error()))
		}

		return nil
	})
	if err != nil {
		return "", errors.Wrap(err, "walk file tree")
	}

	return configuredChartArchive.Name(), nil
}
