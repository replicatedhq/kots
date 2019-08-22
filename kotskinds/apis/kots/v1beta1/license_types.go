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

// LicenseSpec defines the desired state of LicenseSpec
type LicenseSpec struct {
	Signature []byte `json:"signature"`
	AppSlug   string `json:"appSlug"`
	Endpoint  string `json:"endpoint,omitempty"`
	LicenseID string `json:"licenseID"`
}

// LicenseStatus defines the observed state of License
type LicenseStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// License is the Schema for the license API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type License struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LicenseSpec   `json:"spec,omitempty"`
	Status LicenseStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LicenseList contains a list of Licenses
type LicenseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []License `json:"items"`
}

func init() {
	SchemeBuilder.Register(&License{}, &LicenseList{})
}
