package kotsutil

import (
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	"github.com/replicatedhq/kotskinds/pkg/helmchart"
)

// HelmChartValues applies the shared values/optionalValues contract of both
// HelmChart versions, without mutating the chart's configured values.
func HelmChartValues(chart helmchart.HelmChartInterface) (map[string]interface{}, error) {
	data, err := json.Marshal(chart)
	if err != nil {
		return nil, err
	}
	var normalized kotsv1beta2.HelmChart
	if err := json.Unmarshal(data, &normalized); err != nil {
		return nil, err
	}
	merged := normalized.Spec.Values
	if merged == nil {
		merged = map[string]kotsv1beta2.MappedChartValue{}
	}
	for _, optional := range normalized.Spec.OptionalValues {
		enabled, err := strconv.ParseBool(optional.When)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse when conditional on optional values")
		}
		if !enabled {
			continue
		}
		if optional.RecursiveMerge {
			merged = kotsv1beta2.MergeHelmChartValues(merged, optional.Values)
		} else {
			for key, value := range optional.Values {
				merged[key] = value
			}
		}
	}
	return normalized.Spec.GetHelmValues(merged)
}
