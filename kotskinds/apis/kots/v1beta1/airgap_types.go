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

// AirgapSpec defines the desired state of AirgapSpec
type AirgapSpec struct {
	VersionLabel string `json:"versionLabel,omitempty"`
	ReleaseNotes string `json:"releaseNotes,omitempty"`
	Signature    []byte `json:"signature,omitempty"`
}

// AirgapStatus defines the observed state of Airgap
type AirgapStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Airgap is the Schema for the airgap API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Airgap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AirgapSpec   `json:"spec,omitempty"`
	Status AirgapStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AirgapList contains a list of Airgaps
type AirgapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Airgap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Airgap{}, &AirgapList{})
}
