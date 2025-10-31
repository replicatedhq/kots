package validators

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/lint/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	"github.com/replicatedhq/kotskinds/pkg/helmchart"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/client-go/kubernetes/scheme"
)

// ValidateHelmCharts validates that HelmCharts have matching .tgz files and vice versa
func ValidateHelmCharts(renderedFiles, tarGzFiles types.SpecFiles) ([]types.LintExpression, error) {
	lintExpressions := []types.LintExpression{}

	// separate multi docs because the manifest can be a part of a multi doc yaml file
	separatedSpecFiles, err := renderedFiles.Separate()
	if err != nil {
		return nil, errors.Wrap(err, "failed to separate multi docs")
	}

	// check if all helm charts have corresponding archives
	allKotsHelmCharts := findAllKotsHelmCharts(separatedSpecFiles)
	for _, helmChart := range allKotsHelmCharts {
		archiveExists, err := archiveForHelmChartExists(tarGzFiles, helmChart)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if archive for helm chart exists")
		}

		if !archiveExists {
			lintExpression := types.LintExpression{
				Rule:    "helm-archive-missing",
				Type:    "error",
				Message: fmt.Sprintf("Could not find helm archive for chart '%s' version '%s'", helmChart.GetChartName(), helmChart.GetChartVersion()),
			}
			lintExpressions = append(lintExpressions, lintExpression)
		}
	}

	// check if all archives have corresponding helm chart manifests
	for _, specFile := range tarGzFiles {
		if !specFile.IsTarGz() {
			continue
		}

		chartExists, err := helmChartForArchiveExists(allKotsHelmCharts, specFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if helm chart for archive exists")
		}

		if !chartExists {
			lintExpression := types.LintExpression{
				Rule:    "helm-chart-missing",
				Type:    "error",
				Message: fmt.Sprintf("Could not find helm chart manifest for archive '%s'", specFile.Path),
			}
			lintExpressions = append(lintExpressions, lintExpression)
		}
	}

	return lintExpressions, nil
}

// archiveForHelmChartExists iterates through all files, looking for a helm chart archive
// that matches the chart name and version specified in the kotsHelmChart parameter
func archiveForHelmChartExists(specFiles types.SpecFiles, kotsHelmChart helmchart.HelmChartInterface) (bool, error) {
	for _, specFile := range specFiles {
		if !specFile.IsTarGz() {
			continue
		}

		// We treat all .tar.gz archives as helm charts
		files, err := types.SpecFilesFromTarGz(specFile)
		if err != nil {
			return false, errors.Wrap(err, "failed to read chart archive")
		}

		for _, file := range files {
			if file.Path == "Chart.yaml" {
				chartManifest := new(chart.Metadata)
				if err := yaml.Unmarshal([]byte(file.Content), chartManifest); err != nil {
					return false, errors.Wrap(err, "failed to unmarshal chart yaml")
				}

				if chartManifest.Name == kotsHelmChart.GetChartName() {
					if chartManifest.Version == kotsHelmChart.GetChartVersion() {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// helmChartForArchiveExists iterates through all existing helm charts, looking for a helm chart manifest
// that matches the chart name and version specified in the Chart.yaml file in the archive
func helmChartForArchiveExists(allKotsHelmCharts []helmchart.HelmChartInterface, archive types.SpecFile) (bool, error) {
	files, err := types.SpecFilesFromTarGz(archive)
	if err != nil {
		return false, errors.Wrap(err, "failed to read chart archive")
	}

	for _, file := range files {
		if file.Path != "Chart.yaml" {
			continue
		}

		chartManifest := new(chart.Metadata)
		if err := yaml.Unmarshal([]byte(file.Content), chartManifest); err != nil {
			return false, errors.Wrap(err, "failed to unmarshal chart yaml")
		}

		for _, kotsHelmChart := range allKotsHelmCharts {
			if chartManifest.Name == kotsHelmChart.GetChartName() {
				if chartManifest.Version == kotsHelmChart.GetChartVersion() {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func findAllKotsHelmCharts(specFiles types.SpecFiles) []helmchart.HelmChartInterface {
	kotsHelmCharts := []helmchart.HelmChartInterface{}
	for _, specFile := range specFiles {
		kotsHelmChart := tryParsingAsHelmChartGVK([]byte(specFile.Content))
		if kotsHelmChart != nil {
			kotsHelmCharts = append(kotsHelmCharts, kotsHelmChart)
		}
	}

	return kotsHelmCharts
}

func tryParsingAsHelmChartGVK(content []byte) helmchart.HelmChartInterface {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil
	}

	if gvk.Group == "kots.io" {
		if gvk.Version == "v1beta1" {
			if gvk.Kind == "HelmChart" {
				return obj.(*kotsv1beta1.HelmChart)
			}
		} else if gvk.Version == "v1beta2" {
			if gvk.Kind == "HelmChart" {
				return obj.(*kotsv1beta2.HelmChart)
			}
		}
	}

	return nil
}
