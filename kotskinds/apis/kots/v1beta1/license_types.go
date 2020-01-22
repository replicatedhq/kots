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
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Int Type = iota
	String
	Bool
)

type Type int

type EntitlementValue struct {
	Type    Type
	IntVal  int64
	StrVal  string
	BoolVal bool
}

func (entitlementValue *EntitlementValue) Value() interface{} {
	if entitlementValue.Type == Int {
		return entitlementValue.IntVal
	} else if entitlementValue.Type == Bool {
		return entitlementValue.BoolVal
	}

	return entitlementValue.StrVal
}

func (entitlementValue *EntitlementValue) MarshalJSON() ([]byte, error) {
	switch entitlementValue.Type {
	case Int:
		return json.Marshal(entitlementValue.IntVal)

	case String:
		return json.Marshal(entitlementValue.StrVal)

	case Bool:
		return json.Marshal(entitlementValue.BoolVal)

	default:
		return []byte{}, fmt.Errorf("impossible EntitlementValue.Type")
	}
}

func (entitlementValue *EntitlementValue) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		entitlementValue.Type = String
		return json.Unmarshal(value, &entitlementValue.StrVal)
	}

	intValue, err := strconv.ParseInt(string(value), 10, 64)
	if err == nil {
		entitlementValue.Type = Int
		entitlementValue.IntVal = intValue
		return nil
	}

	boolValue, err := strconv.ParseBool(string(value))
	if err == nil {
		entitlementValue.Type = Bool
		entitlementValue.BoolVal = boolValue
		return nil
	}

	return errors.New("unknown license value type")
}

type EntitlementField struct {
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description,omitempty"`
	Value       EntitlementValue `json:"value,omitempty"`
	ValueType   string           `json:"valueType,omitempty"`
	IsHidden    bool             `json:"isHidden,omitempty"`
}

// LicenseSpec defines the desired state of LicenseSpec
type LicenseSpec struct {
	Signature           []byte                      `json:"signature"`
	AppSlug             string                      `json:"appSlug"`
	Endpoint            string                      `json:"endpoint,omitempty"`
	CustomerName        string                      `json:"customerName,omitempty"`
	ChannelName         string                      `json:"channelName,omitempty"`
	LicenseSequence     int64                       `json:"licenseSequence,omitempty"`
	LicenseID           string                      `json:"licenseID"`
	LicenseType         string                      `json:"licenseType,omitempty"`
	IsAirgapSupported   bool                        `json:"isAirgapSupported,omitempty"`
	IsGitOpsSupported   bool                        `json:"isGitOpsSupported,omitempty"`
	IsSnapshotSupported bool                        `json:"isSnapshotSupported,omitempty"`
	Entitlements        map[string]EntitlementField `json:"entitlements,omitempty"`
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
