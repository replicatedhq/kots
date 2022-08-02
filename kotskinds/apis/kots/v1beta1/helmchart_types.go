/*
Copyright 2019 Replicated, Inc..

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotskinds/multitype"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Type=""
type MappedChartValue struct {
	Value string `json:"-"`

	valueType string `json:"-"`

	strValue   string  `json:"-"`
	boolValue  bool    `json:"-"`
	floatValue float64 `json:"-"`

	children map[string]*MappedChartValue `json:"-"`
	array    []*MappedChartValue          `json:"-"`
}

func (m *MappedChartValue) getBuiltValue() (interface{}, error) {
	if m.valueType == "string" {
		return m.strValue, nil
	}
	if m.valueType == "bool" {
		return m.boolValue, nil
	}
	if m.valueType == "float" {
		return m.floatValue, nil
	}
	if m.valueType == "nil" {
		return nil, nil
	}

	if m.valueType == "children" {
		children := map[string]interface{}{}
		for k, v := range m.children {
			childValue, err := v.getBuiltValue()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get value of child %s", k)
			}
			children[k] = childValue
		}
		return children, nil
	}
	if m.valueType == "array" {
		var elements []interface{}
		for i, v := range m.array {
			elValue, err := v.getBuiltValue()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get value of child %d", i)
			}
			elements = append(elements, elValue)
		}
		return elements, nil
	}

	return nil, errors.New("unknown value type")
}

func (m *MappedChartValue) UnmarshalJSON(value []byte) error {
	var b interface{}
	err := json.Unmarshal(value, &b)
	if err != nil {
		return err
	}

	if b == nil {
		m.valueType = "nil"
		return nil
	}

	if b, ok := b.(string); ok {
		m.strValue = b
		m.valueType = "string"
		return nil
	}

	if b, ok := b.(bool); ok {
		m.boolValue = b
		m.valueType = "bool"
		return nil
	}

	if b, ok := b.(float64); ok {
		m.floatValue = b
		m.valueType = "float"
		return nil
	}

	if b, ok := b.(map[string]interface{}); ok {
		m.children = make(map[string]*MappedChartValue)
		for k, v := range b {
			vv, err := json.Marshal(v)
			if err != nil {
				return err
			}

			m2 := &MappedChartValue{}
			if err := m2.UnmarshalJSON(vv); err != nil {
				return err
			}

			m.children[k] = m2
		}

		m.valueType = "children"

		return nil
	}

	if b, ok := b.([]interface{}); ok {
		m.array = []*MappedChartValue{}
		for _, v := range b {
			vv, err := json.Marshal(v)
			if err != nil {
				return err
			}

			m2 := &MappedChartValue{}
			if err := m2.UnmarshalJSON(vv); err != nil {
				return err
			}

			m.array = append(m.array, m2)
		}

		m.valueType = "array"

		return nil
	}

	return errors.Errorf("unknown mapped chart value type: %T", b)
}

type ChartIdentifier struct {
	Name         string `json:"name"`
	ChartVersion string `json:"chartVersion"`
	ReleaseName  string `json:"releaseName,omitempty"`
}

func (h *HelmChartSpec) GetHelmValues(values map[string]MappedChartValue) (map[string]interface{}, error) {
	result := map[string]interface{}{}

	for k, v := range values {
		value, err := h.renderValue(&v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to render value at %s", k)
		}

		result[k] = value
	}

	return result, nil
}

func (h *HelmChartSpec) GetReplTmplValues(values map[string]MappedChartValue) (map[string]interface{}, error) {
	newValues := make(map[string]interface{})

	for k, v := range values {
		value, err := h.getReplTmplValue(&v)
		if err != nil || value == nil {
			continue
		}
		newValues[k] = value
	}

	return newValues, nil
}

func (h *HelmChartSpec) getReplTmplValue(value *MappedChartValue) (interface{}, error) {
	if value.valueType == "children" {
		result := map[string]interface{}{}
		for k, v := range value.children {
			built, err := h.getReplTmplValue(v)
			if err != nil || built == nil {
				continue
			}
			result[k] = built
		}
		if len(result) == 0 {
			return nil, nil
		}
		return result, nil
	} else if value.valueType == "array" {
		result := []interface{}{}
		for _, v := range value.array {
			built, err := h.getReplTmplValue(v)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render array value")
			}
			result = append(result, built)
		}
		return result, nil
	} else {
		built, err := value.getBuiltValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to build value")
		}
		str, ok := built.(string)
		if ok && (strings.Contains(str, "repl{{") || strings.Contains(str, "{{repl")) {
			return built, nil
		}
		return nil, errors.New("value is not string or not repl tmpl function")
	}
}

func GetMapIntersect(m1, m2 map[string]interface{}) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	for k, v := range m1 {
		res[k] = intersect(v, k, m2)
	}

	return res, nil
}

func intersect(v interface{}, k string, m2 map[string]interface{}) interface{} {
	switch v.(type) {
	case string, int, bool:
		if v2, ok := m2[k]; ok {
			return v2
		}
	case map[string]interface{}:
		for k2, v2 := range v.(map[string]interface{}) {
			_, ok := m2[k].(map[string]interface{})
			if ok {
				val := intersect(v2, k2, m2[k].(map[string]interface{}))
				// set map here so only tmpl keys are present in returned result
				m2[k] = v.(map[string]interface{})
				// set child map value from intersection
				m2[k].(map[string]interface{})[k2] = val
				return m2[k]
			}
		}
	default:
		return nil
	}

	return nil
}

func MergeHelmChartValues(baseValues map[string]MappedChartValue,
	overlayValues map[string]MappedChartValue) map[string]MappedChartValue {

	result := map[string]MappedChartValue{}
	for k, v := range baseValues {
		if _, exists := overlayValues[k]; !exists {
			result[k] = baseValues[k]
			continue
		}
		if v.valueType != "children" {
			result[k] = overlayValues[k]
		} else {
			result[k] = MappedChartValue{
				valueType: "children",
				children:  mergeValueChildren(v.children, overlayValues[k].children),
			}
		}
	}
	for k, v := range overlayValues {
		if _, exists := baseValues[k]; !exists {
			result[k] = v
		}
	}
	return result
}

func mergeValueChildren(baseValues map[string]*MappedChartValue, overlayValues map[string]*MappedChartValue) map[string]*MappedChartValue {
	result := map[string]*MappedChartValue{}
	for k, v := range baseValues {
		if _, exists := overlayValues[k]; !exists {
			result[k] = baseValues[k]
			continue
		}
		if v.valueType != "children" {
			result[k] = overlayValues[k]
		} else {
			result[k] = &MappedChartValue{
				valueType: "children",
				children:  mergeValueChildren(v.children, overlayValues[k].children),
			}
		}
	}

	for k, v := range overlayValues {
		if _, exists := baseValues[k]; !exists {
			result[k] = v
		}
	}
	return result
}

func (h *HelmChartSpec) renderValue(value *MappedChartValue) (interface{}, error) {
	if value.valueType == "children" {
		result := map[string]interface{}{}
		for k, v := range value.children {
			built, err := h.renderValue(v)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to render child value at key %s", k)
			}
			result[k] = built
		}
		return result, nil
	} else if value.valueType == "array" {
		result := []interface{}{}
		for _, v := range value.array {
			built, err := h.renderValue(v)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render array value")
			}
			result = append(result, built)
		}
		return result, nil
	} else {
		built, err := value.getBuiltValue()
		if err != nil {
			return nil, errors.Wrap(err, "failed to build value")
		}
		return built, nil
	}
}

func (h *HelmChart) GetDirName() string {
	if h.Spec.Chart.ReleaseName != "" {
		return h.Spec.Chart.ReleaseName
	}
	return h.Name
}

func (h *HelmChart) GetReleaseName() string {
	if h.Spec.Chart.ReleaseName != "" {
		return h.Spec.Chart.ReleaseName
	}
	return h.Spec.Chart.Name
}

type OptionalValue struct {
	When           string `json:"when"`
	RecursiveMerge bool   `json:"recursiveMerge"`

	Values map[string]MappedChartValue `json:"values,omitempty"`
}

// HelmChartSpec defines the desired state of HelmChartSpec
type HelmChartSpec struct {
	Chart            ChartIdentifier             `json:"chart"`
	Exclude          multitype.BoolOrString      `json:"exclude,omitempty"`
	HelmVersion      string                      `json:"helmVersion,omitempty"`
	UseHelmInstall   bool                        `json:"useHelmInstall,omitempty"`
	Namespace        string                      `json:"namespace,omitempty"`
	Values           map[string]MappedChartValue `json:"values,omitempty"`
	OptionalValues   []*OptionalValue            `json:"optionalValues,omitempty"`
	Builder          map[string]MappedChartValue `json:"builder,omitempty"`
	Weight           int64                       `json:"weight,omitempty"`
	HelmUpgradeFlags []string                    `json:"helmUpgradeFlags,omitempty"`
}

// HelmChartStatus defines the observed state of HelmChart
type HelmChartStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// HelmChart is the Schema for the helmchart API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type HelmChart struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmChartSpec   `json:"spec,omitempty"`
	Status HelmChartStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HelmChartList contains a list of HelmCharts
type HelmChartList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HelmChart `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelmChart{}, &HelmChartList{})
}
