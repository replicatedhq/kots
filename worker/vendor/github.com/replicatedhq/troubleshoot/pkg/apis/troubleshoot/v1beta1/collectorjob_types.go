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

type CollectorRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// CollectorJobSpec defines the desired state of CollectorJob
type CollectorJobSpec struct {
	Collector CollectorRef `json:"collector"`

	Image           string `json:"image,omitempty"`
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`
	Redact          bool   `json:"redact,omitempty"`
}

// CollectorJobStatus defines the observed state of CollectorJob
type CollectorJobStatus struct {
	IsServerReady      bool   `json:"isServerReady"`
	ServerPodName      string `json:"serverPodName"`
	ServerPodNamespace string `json:"serverPodNamespace"`
	ServerPodPort      int    `json:"serverPodPort"`

	Running    []string `json:"running"`
	Successful []string `json:"successful"`
	Failed     []string `json:"failed"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// CollectorJob is the Schema for the collectorjobs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type CollectorJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CollectorJobSpec   `json:"spec,omitempty"`
	Status CollectorJobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CollectorJobList contains a list of CollectorJob
type CollectorJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CollectorJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CollectorJob{}, &CollectorJobList{})
}
