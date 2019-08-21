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

type ResultRequest struct {
	URI    string `json:"uri" yaml:"uri"`
	Method string `json:"method" yaml:"method"`
}

type AfterCollection struct {
	UploadResultsTo *ResultRequest `json:"uploadResultsTo,omitempty" yaml:"uploadResultsTo,omitempty"`
	Callback        *ResultRequest `json:"callback,omitempty" yaml:"callback,omitempty"`
}

// CollectorSpec defines the desired state of Collector
type CollectorSpec struct {
	Collectors      []*Collect         `json:"collectors,omitempty" yaml:"collectors,omitempty"`
	AfterCollection []*AfterCollection `json:"afterCollection,omitempty" yaml:"afterCollection,omitempty"`
}

// CollectorStatus defines the observed state of Collector
type CollectorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Collector is the Schema for the collectors API
// +k8s:openapi-gen=true
type Collector struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   CollectorSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status CollectorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CollectorList contains a list of Collector
type CollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Collector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Collector{}, &CollectorList{})
}
