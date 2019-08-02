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

// PreflightSpec defines the desired state of Preflight
type PreflightSpec struct {
	UploadResultsTo string     `json:"uploadResultsTo,omitempty" yaml:"uploadResultsTo,omitempty"`
	Collectors      []*Collect `json:"collectors,omitempty" yaml:"collectors,omitempty"`
	Analyzers       []*Analyze `json:"analyzers,omitempty" yaml:"analyzers,omitempty"`
}

// PreflightStatus defines the observed state of Preflight
type PreflightStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Preflight is the Schema for the preflights API
// +k8s:openapi-gen=true
type Preflight struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   PreflightSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status PreflightStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PreflightList contains a list of Preflight
type PreflightList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Preflight `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Preflight{}, &PreflightList{})
}
