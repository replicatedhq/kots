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

	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MappedChartValue struct {
	Value string `json:",inline"`

	directValue string                      `json:"-"`
	children    map[string]MappedChartValue `json:"-"`
}

func (m *MappedChartValue) GetValue() interface{} {
	if m.directValue != "" {
		return m.directValue
	}

	return m.children
}

func (m *MappedChartValue) UnmarshalJSON(value []byte) error {
	var b interface{}
	err := json.Unmarshal(value, &b)
	if err != nil {
		return err
	}

	if b, ok := b.(string); ok {
		m.directValue = b
		return nil
	}

	if b, ok := b.(map[string]MappedChartValue); ok {
		m.children = b
		return nil
	}

	return errors.Errorf("unknown mapped chart value type: %T", b)
}

type ChartIdentifier struct {
	Name         string `json:"name"`
	ChartVersion string `json:"chartVersion"`
}

// HelmChartSpec defines the desired state of HelmChartSpec
type HelmChartSpec struct {
	Chart   ChartIdentifier             `json:"chart"`
	Values  map[string]MappedChartValue `json:"values,omitempty"`
	Builder map[string]MappedChartValue `json:"builder,omitempty"`
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
