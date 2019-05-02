package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ShipWatch is the Schema for the shipwatches API
// +k8s:openapi-gen=true
type ShipWatch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShipWatchSpec   `json:"spec,omitempty"`
	Status ShipWatchStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ShipWatchList contains a list of ShipWatch
type ShipWatchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ShipWatch `json:"items"`
}

type ShipWatchSpec struct {
	State         StateSpec       `json:"state"`
	Images        []ImageSpec     `json:"images,omitempty"`
	WatchInterval string          `json:"watchInterval,omitempty"`
	Environment   []corev1.EnvVar `json:"environment,omitempty"`
	// +kubebuilder:validation:MinItems=1
	Actions []ActionSpec `json:"actions,omitempty"`
}

type StateSpec struct {
	ValueFrom ShipWatchValueFromSpec `json:"valueFrom,omitempty"`
}

type ImageSpec struct {
	Image           string `json:"image"`
	Tag             string `json:"tag"`
	ImagePullPolicy string `json:"imagePullPolicy"`
}

type ShipWatchValueFromSpec struct {
	SecretKeyRef SecretKeyRef `json:"secretKeyRef,omitempty"`
}

type SecretKeyRef struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	// A service account authorized to read/write this secret
	ServiceAccountName string `json:"serviceAccountName"`
}

// This is going to get an app token that will be exchanged for a JWT
type GitHubRef struct {
	Username string    `json:"username"`
	Owner    string    `json:"owner"`
	Repo     string    `json:"repo"`
	Branch   string    `json:"branch"`
	Path     string    `json:"path,omitempty"`
	Token    TokenSpec `json:"token"`
	Key      KeySpec   `json:"key"`
}

type ActionSpec struct {
	Name string `json:"name"`

	PullRequest *PullRequestActionSpec `json:"pullRequest,omitempty"`
	Webhook     *WebhookActionSpec     `json:"webhook,omitempty"`
}

type PullRequestActionSpec struct {
	Message  string     `json:"message"`
	GitHub   *GitHubRef `json:"github,omitempty"`
	BasePath string     `json:"basePath"`
}

type WebhookActionSpec struct {
	URI     string `json:"uri"`
	Payload string `json:"payload"`
	Secret  string `json:"secret,omitempty"`
}

type KeySpec struct {
	ValueFrom ShipWatchValueFromSpec `json:"valueFrom,omitempty"`
}

type TokenSpec struct {
	ValueFrom ShipWatchValueFromSpec `json:"valueFrom,omitempty"`
}

type ShipWatchStatus struct {
}

func init() {
	SchemeBuilder.Register(&ShipWatch{}, &ShipWatchList{})
}
