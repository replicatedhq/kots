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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigValue struct {
	Default        string `json:"default,omitempty"`
	Value          string `json:"value,omitempty"`
	Data           string `json:"data,omitempty"`
	ValuePlaintext string `json:"valuePlaintext,omitempty"`
	DataPlaintext  string `json:"dataPlaintext,omitempty"`
	Filename       string `json:"filename,omitempty"`
	RepeatableItem string `json:"repeatableItem,omitempty"`
}

// ConfigValuesSpec defines the desired state of ConfigValue
type ConfigValuesSpec struct {
	Values map[string]ConfigValue `json:"values"`
}

// ConfigValuesStatus defines the observed state of ConfigValues
type ConfigValuesStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// App is the Schema for the app API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type ConfigValues struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigValuesSpec   `json:"spec,omitempty"`
	Status ConfigValuesStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApConfigValuesListpList contains a list of ConfigValues
type ConfigValuesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfigValues `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConfigValues{}, &ConfigValuesList{})
}
