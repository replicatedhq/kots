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

type IngressSpec struct {
	Enabled  bool                   `json:"enabled" yaml:"enabled"`
	Ingress  *IngressConfig         `json:"ingress,omitempty" yaml:"ingress,omitempty"`
	NodePort *IngressNodePortConfig `json:"nodePort,omitempty" yaml:"nodePort,omitempty"`
}

type IngressConfig struct {
	Path          string            `json:"path" yaml:"path"`
	Host          string            `json:"host" yaml:"host"`
	TLSSecretName string            `json:"tlsSecretName,omitempty" yaml:"tlsSecretName,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

type IngressNodePortConfig struct {
	Port int `json:"port" yaml:"port"`
}

// IngressStatus defines the observed state of Ingress
type IngressStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Ingress is the Schema for the ingress document
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Ingress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IngressSpec   `json:"spec,omitempty"`
	Status IngressStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IngressList contains a list of Identities
type IngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Ingress `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Ingress{}, &IngressList{})
}
