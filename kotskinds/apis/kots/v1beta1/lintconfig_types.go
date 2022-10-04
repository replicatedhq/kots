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

type LintLevel string

const (
	Error LintLevel = "error"
	Warn  LintLevel = "warn"
	Info  LintLevel = "info"
	Off   LintLevel = "off"
)

type LintRule struct {
	Name  string    `json:"name"`
	Level LintLevel `json:"level,omitempty"`
}

// LintConfigSpec defines the desired state of LintConfig
type LintConfigSpec struct {
	Rules []LintRule `json:"rules"`
}

// LintConfigStatus defines the observed state of LintConfig
type LintConfigStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// LintConfig is the Schema for the lint config API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type LintConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LintConfigSpec   `json:"spec,omitempty"`
	Status LintConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LintConfigList contains a list of Configs
type LintConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LintConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LintConfig{}, &LintConfigList{})
}
