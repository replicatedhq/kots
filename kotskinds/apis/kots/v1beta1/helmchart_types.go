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
	Value string `json:"value,omitempty"`

	ValueType string `json:"valueType,omitempty"`

	StrValue   string  `json:"strValue,omitempty"`
	BoolValue  bool    `json:"boolValue,omitempty"`
	FloatValue float64 `json:"floatValue,omitempty"`

	Children map[string]*MappedChartValue `json:"children,omitempty"`
	Array    []*MappedChartValue          `json:"array,omitempty"`
}

func (m *MappedChartValue) getBuiltValue() (interface{}, error) {
	if m.ValueType == "string" {
		return m.StrValue, nil
	}
	if m.ValueType == "bool" {
		return m.BoolValue, nil
	}
	if m.ValueType == "float" {
		return m.FloatValue, nil
	}
	if m.ValueType == "nil" {
		return nil, nil
	}

	if m.ValueType == "children" {
		children := map[string]interface{}{}
		for k, v := range m.Children {
			childValue, err := v.getBuiltValue()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get value of child %s", k)
			}
			children[k] = childValue
		}
		return children, nil
	}
	if m.ValueType == "array" {
		var elements []interface{}
		for i, v := range m.Array {
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
		m.ValueType = "nil"
		return nil
	}

	if b, ok := b.(string); ok {
		m.StrValue = b
		m.ValueType = "string"
		return nil
	}

	if b, ok := b.(bool); ok {
		m.BoolValue = b
		m.ValueType = "bool"
		return nil
	}

	if b, ok := b.(float64); ok {
		m.FloatValue = b
		m.ValueType = "float"
		return nil
	}

	if b, ok := b.(map[string]interface{}); ok {
		m.Children = make(map[string]*MappedChartValue)
		for k, v := range b {
			vv, err := json.Marshal(v)
			if err != nil {
				return err
			}

			m2 := &MappedChartValue{}
			if err := m2.UnmarshalJSON(vv); err != nil {
				return err
			}

			m.Children[k] = m2
		}

		m.ValueType = "children"

		return nil
	}

	if b, ok := b.([]interface{}); ok {
		m.Array = []*MappedChartValue{}
		for _, v := range b {
			vv, err := json.Marshal(v)
			if err != nil {
				return err
			}

			m2 := &MappedChartValue{}
			if err := m2.UnmarshalJSON(vv); err != nil {
				return err
			}

			m.Array = append(m.Array, m2)
		}

		m.ValueType = "array"

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
	if value.ValueType == "children" {
		result := map[string]interface{}{}
		for k, v := range value.Children {
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
	} else if value.ValueType == "array" {
		result := []interface{}{}
		for _, v := range value.Array {
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

func GetMapIntersect(m1, m2 map[string]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v1 := range m1 {
		v2, ok := m2[k]
		if !ok {
			continue
		}

		v1map, v1mapOK := v1.(map[string]interface{})
		v2map, v2mapOK := v2.(map[string]interface{})
		if v1mapOK && v2mapOK {
			res[k] = GetMapIntersect(v1map, v2map)
			continue
		}

		res[k] = v2
	}

	return res
}

func MergeHelmChartValues(baseValues map[string]MappedChartValue,
	overlayValues map[string]MappedChartValue) map[string]MappedChartValue {

	result := map[string]MappedChartValue{}
	for k, v := range baseValues {
		if _, exists := overlayValues[k]; !exists {
			result[k] = baseValues[k]
			continue
		}
		if v.ValueType != "children" {
			result[k] = overlayValues[k]
		} else {
			result[k] = MappedChartValue{
				ValueType: "children",
				Children:  mergeValueChildren(v.Children, overlayValues[k].Children),
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
		if v.ValueType != "children" {
			result[k] = overlayValues[k]
		} else {
			result[k] = &MappedChartValue{
				ValueType: "children",
				Children:  mergeValueChildren(v.Children, overlayValues[k].Children),
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
	if value.ValueType == "children" {
		result := map[string]interface{}{}
		for k, v := range value.Children {
			built, err := h.renderValue(v)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to render child value at key %s", k)
			}
			result[k] = built
		}
		return result, nil
	} else if value.ValueType == "array" {
		result := []interface{}{}
		for _, v := range value.Array {
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
