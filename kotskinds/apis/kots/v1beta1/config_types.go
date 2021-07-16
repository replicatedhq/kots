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

	"github.com/replicatedhq/kots/kotskinds/multitype"
)

type ConfigChildItem struct {
	Name        string                 `json:"name"`
	Title       string                 `json:"title"`
	Recommended bool                   `json:"recommended,omitempty"`
	Default     multitype.BoolOrString `json:"default,omitempty"`
	Value       multitype.BoolOrString `json:"value,omitempty"`
}

type ConfigItem struct {
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Title         string                 `json:"title,omitempty"`
	HelpText      string                 `json:"help_text,omitempty"`
	Recommended   bool                   `json:"recommended,omitempty"`
	Default       multitype.BoolOrString `json:"default,omitempty"`
	Value         multitype.BoolOrString `json:"value,omitempty"`
	Data          string                 `json:"data,omitempty"`
	Error         string                 `json:"error,omitempty"`
	MultiValue    []string               `json:"multi_value,omitempty"`
	ReadOnly      bool                   `json:"readonly,omitempty"`
	WriteOnce     bool                   `json:"write_once,omitempty"`
	When          multitype.QuotedBool   `json:"when,omitempty"`
	Multiple      bool                   `json:"multiple,omitempty"`
	Hidden        bool                   `json:"hidden,omitempty"`
	Position      int                    `json:"-"`
	Affix         string                 `json:"affix,omitempty"`
	Required      bool                   `json:"required,omitempty"`
	Items         []ConfigChildItem      `json:"items,omitempty"`
	Filename      string                 `json:"filename,omitempty"`
	Repeatable    bool                   `json:"repeatable,omitempty"`
	MinimumCount  int                    `json:"minimumCount,omitempty"`
	CountByGroup  map[string]int         `json:"countByGroup,omitempty"`
	Templates     []RepeatTemplate       `json:"templates,omitempty"`
	ValuesByGroup ValuesByGroup          `json:"valuesByGroup,omitempty"`
	// Props       map[string]interface{} `json:"props,omitempty"`
	// DefaultCmd  *ConfigItemCmd         `json:"default_cmd,omitempty"`
	// ValueCmd    *ConfigItemCmd         `json:"value_cmd,omitempty"`
	// DataCmd     *ConfigItemCmd         `json:"data_cmd,omitempty"`
}

type RepeatTemplate struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	YamlPath   string `json:"yamlPath,omitempty"`
}

type ValuesByGroup map[string]GroupValues

type GroupValues map[string]string

type ConfigGroup struct {
	Name        string               `json:"name"`
	Title       string               `json:"title"`
	Description string               `json:"description,omitempty"`
	When        multitype.QuotedBool `json:"when,omitempty"`
	Items       []ConfigItem         `json:"items,omitempty"`
}

// ConfigSpec defines the desired state of ConfigSpec
type ConfigSpec struct {
	Groups []ConfigGroup `json:"groups"`
}

// ConfigStatus defines the observed state of Config
type ConfigStatus struct {
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Config is the Schema for the config API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigSpec   `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfigList contains a list of Configs
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Config `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Config{}, &ConfigList{})
}
