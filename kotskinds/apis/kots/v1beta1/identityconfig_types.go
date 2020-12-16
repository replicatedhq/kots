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
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

type IdentityConfigSpec struct {
	Enabled                bool                    `json:"enabled" yaml:"enabled"`
	DisablePasswordAuth    bool                    `json:"disablePasswordAuth,omitempty" yaml:"disablePasswordAuth,omitempty"`
	Groups                 []IdentityConfigGroup   `json:"groups,omitempty" yaml:"groups,omitempty"`
	IngressConfig          IngressConfigSpec       `json:"ingressConfig,omitempty" yaml:"ingressConfig,omitempty"`
	AdminConsoleAddress    string                  `json:"adminConsoleAddress,omitempty" yaml:"adminConsoleAddress,omitempty"` // TODO (ethan): this does not belong here
	IdentityServiceAddress string                  `json:"identityServiceAddress,omitempty" yaml:"identityServiceAddress,omitempty"`
	CACertPemBase64        string                  `json:"caCertPemBase64,omitempty" yaml:"caCertPemBase64,omitempty"`
	InsecureSkipTLSVerify  bool                    `json:"insecureSkipTLSVerify,omitempty" yaml:"insecureSkipTLSVerify,omitempty"`
	Storage                Storage                 `json:"storage,omitempty" yaml:"storage,omitempty"`
	ClientID               string                  `json:"clientID,omitempty" yaml:"clientID,omitempty"`
	ClientSecret           *StringValueOrEncrypted `json:"clientSecret,omitempty" yaml:"clientSecret,omitempty"`
	DexConnectors          DexConnectors           `json:"dexConnectors,omitempty" yaml:"dexConnectors,omitempty"`
}

type StringValueOrEncrypted struct {
	Value          string `json:"value,omitempty" yaml:"value,omitempty"`
	ValueEncrypted string `json:"valueEncrypted,omitempty" yaml:"valueEncrypted,omitempty"`
}

func NewStringValueOrEncrypted(value string, cipher crypto.AESCipher) *StringValueOrEncrypted {
	v := &StringValueOrEncrypted{Value: value}
	v.EncryptValue(cipher)
	return v
}

func (v *StringValueOrEncrypted) GetValue(cipher crypto.AESCipher) (string, error) {
	if v == nil {
		return "", nil
	}
	if v.ValueEncrypted != "" {
		b, err := base64.StdEncoding.DecodeString(v.ValueEncrypted)
		if err != nil {
			return "", errors.Wrap(err, "failed to base64 decode")
		}
		result, err := cipher.Decrypt(b)
		return string(result), errors.Wrap(err, "failed to decrypt")
	}
	return v.Value, nil
}

func (v *StringValueOrEncrypted) EncryptValue(cipher crypto.AESCipher) {
	if v.ValueEncrypted != "" && v.Value == "" {
		return
	}
	v.ValueEncrypted = base64.StdEncoding.EncodeToString(cipher.Encrypt([]byte(v.Value)))
	v.Value = ""
}

type Storage struct {
	PostgresConfig *IdentityPostgresConfig `json:"postgresConfig,omitempty" yaml:"postgresConfig,omitempty"`
}

type IdentityPostgresConfig struct {
	Host     string                  `json:"host" yaml:"host"`
	Port     string                  `json:"port,omitempty" yaml:"port,omitempty"`
	Database string                  `json:"database" yaml:"database"`
	User     string                  `json:"user" yaml:"user"`
	Password *StringValueOrEncrypted `json:"password" yaml:"password"`
}

type IdentityConfigGroup struct {
	ID      string   `json:"id" yaml:"id"`
	RoleIDs []string `json:"roleIds" yaml:"roleIds"`
}

type DexConnectors struct {
	Value          []DexConnector       `json:"value,omitempty" yaml:"value,omitempty"`
	ValueEncrypted string               `json:"valueEncrypted,omitempty" yaml:"valueEncrypted,omitempty"`
	ValueFrom      *DexConnectorsSource `json:"valueFrom,omitempty" yaml:"valueFrom,omitempty"`
}

func (v *DexConnectors) GetValue(cipher crypto.AESCipher) ([]DexConnector, error) {
	if v.ValueEncrypted != "" {
		b, err := base64.StdEncoding.DecodeString(v.ValueEncrypted)
		if err != nil {
			return nil, errors.Wrap(err, "failed to base64 decode")
		}
		result, err := cipher.Decrypt(b)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decrypt")
		}
		if len(result) == 0 {
			return nil, nil
		}
		err = json.Unmarshal(result, &v.Value)
		return v.Value, errors.Wrap(err, "failed to json unmarshal")
	}
	return v.Value, nil
}

func (v *DexConnectors) EncryptValue(cipher crypto.AESCipher) error {
	if v.ValueEncrypted != "" && len(v.Value) == 0 {
		return nil
	}

	b, err := json.Marshal(v.Value)
	if err != nil {
		return err
	}
	v.ValueEncrypted = base64.StdEncoding.EncodeToString(cipher.Encrypt(b))
	v.Value = nil
	return nil
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

// IdentityConfigStatus defines the observed state of IdentityConfig
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
