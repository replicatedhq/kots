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

type PreflightRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// PreflightJobSpec defines the desired state of PreflightJob
type PreflightJobSpec struct {
	Preflight PreflightRef `json:"preflight"`

	Image                    string `json:"preflightImage,omitempty"`
	ImagePullPolicy          string `json:"imagePullPolicy,omitempty"`
	CollectorImage           string `json:"collectorImage,omitempty"`
	CollectorImagePullPolicy string `json:"collectorImagePullPolicy,omitempty"`
}

// PreflightJobStatus defines the observed state of PreflightJob
type PreflightJobStatus struct {
	IsServerReady      bool   `json:"isServerReady"`
	ServerPodName      string `json:"serverPodName"`
	ServerPodNamespace string `json:"serverPodNamespace"`
	ServerPodPort      int    `json:"serverPodPort"`

	CollectorsRunning    []string `json:"collectorsRunning"`
	CollectorsSuccessful []string `json:"collectorsSuccessful"`
	CollectorsFailed     []string `json:"collectorsFailed"`

	IsAnalyzersComplete bool     `json:"isAnalyzersComplete"`
	AnalyzersRunning    []string `json:"analyzersRunning"`
	AnalyzersSuccessful []string `json:"analyzersSuccessful"`
	AnalyzersFailed     []string `json:"analyzersFailed"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PreflightJob is the Schema for the preflightjobs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type PreflightJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PreflightJobSpec   `json:"spec,omitempty"`
	Status PreflightJobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PreflightJobList contains a list of PreflightJob
type PreflightJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PreflightJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PreflightJob{}, &PreflightJobList{})
}
