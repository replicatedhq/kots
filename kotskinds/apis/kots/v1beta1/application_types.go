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

type ApplicationPort struct {
	ServiceName    string `json:"serviceName"`
	ServicePort    int    `json:"servicePort"`
	LocalPort      int    `json:"localPort,omitempty"`
	ApplicationURL string `json:"applicationUrl,omitempty"`
}

// ApplicationSpec defines the desired state of ApplicationSpec
type ApplicationSpec struct {
	Title            string            `json:"title"`
	Icon             string            `json:"icon,omitempty"`
	ApplicationPorts []ApplicationPort `json:"ports,omitempty"`
	ReleaseNotes     string            `json:"releaseNotes,omitempty"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Application is the Schema for the application API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApplicationList contains a list of Applications
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
