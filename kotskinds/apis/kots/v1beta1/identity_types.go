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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type IdentityConfigSpec struct {
	Enabled                bool              `json:"enabled" yaml:"enabled"`
	DisablePasswordAuth    bool              `json:"disablePasswordAuth,omitempty" yaml:"disablePasswordAuth,omitempty"`
	Groups                 []IdentityGroup   `json:"groups,omitempty" yaml:"groups,omitempty"`
	IngressConfig          IngressConfigSpec `json:"ingressConfig,omitempty" yaml:"ingressConfig,omitempty"`
	AdminConsoleAddress    string            `json:"adminConsoleAddress,omitempty" yaml:"adminConsoleAddress,omitempty"`
	IdentityServiceAddress string            `json:"identityServiceAddress,omitempty" yaml:"identityServiceAddress,omitempty"`
	CACertPemBase64        string            `json:"caCertPemBase64,omitempty" yaml:"caCertPemBase64,omitempty"`
	InsecureSkipTLSVerify  bool              `json:"insecureSkipTLSVerify,omitempty" yaml:"insecureSkipTLSVerify,omitempty"`
	DexConnectors          DexConnectors     `json:"dexConnectors,omitempty" yaml:"dexConnectors,omitempty"`
}

type IdentityGroup struct {
	ID      string   `json:"id" yaml:"id"`
	RoleIDs []string `json:"roleIds" yaml:"roleIds"`
}

type DexConnectors struct {
	Value     []DexConnector       `json:"value,omitempty" yaml:"value,omitempty"`
	ValueFrom *DexConnectorsSource `json:"valueFrom,omitempty" yaml:"valueFrom,omitempty"`
}

type DexConnectorsSource struct {
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty" yaml:"secretKeyRef,omitempty"`
}

type DexConnector struct {
	Type string `json:"type"`
	Name string `json:"name"`
	ID   string `json:"id"`

	Config runtime.RawExtension `json:"config"`
}

// IdentityConfigStatus defines the observed state of Identity
type IdentityConfigStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// IdentityConfig is the Schema for the identity config document
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type IdentityConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IdentityConfigSpec   `json:"spec,omitempty"`
	Status IdentityConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IdentityConfigList contains a list of IdentityConfigs
type IdentityConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IdentityConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IdentityConfig{}, &IdentityConfigList{})
}
