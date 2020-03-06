// Based on https://github.com/kubernetes/apimachinery/blob/455a99f/pkg/util/intstr/intstr.go

package multitype

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// QuotedBool is a string type that can also unmarshal raw yaml bools.
//
// +protobuf=true
// +protobuf.options.(gogoproto.goproto_stringer)=false
// +k8s:openapi-gen=true
// +kubebuilder:validation:Type=QuotedBool
type QuotedBool string

// UnmarshalJSON implements the json.Unmarshaller interface.
func (b *QuotedBool) UnmarshalJSON(value []byte) error {
	trueValues := []string{"y", "Y", "yes", "Yes", "YES", "true", "True", "TRUE", "on", "On", "ON", "1"}
	falseValues := []string{"n", "N", "no", "No", "NO", "false", "False", "FALSE", "off", "Off", "OFF", "0"}
	for _, v := range trueValues {
		if string(value) == v {
			*b = "true"
			return nil
		}
	}
	for _, v := range falseValues {
		if string(value) == v {
			*b = "false"
			return nil
		}
	}
	var s string
	if err := json.Unmarshal(value, &s); err != nil {
		return errors.Wrapf(err, "unable to unmarshal %q as bool or string", string(value))
	}
	*b = QuotedBool(s)
	return nil
}

// UnmarshalJSON implements the yaml.Unmarshaller interface.
func (b *QuotedBool) UnmarshalYAML(unmarshal func(interface{}) error) error {
	boolTry := false
	intTry := 0
	stringTry := ""
	err := unmarshal(&boolTry)
	if err == nil {
		if boolTry {
			*b = "true"
		} else {
			*b = "false"
		}
		return nil
	}

	err = unmarshal(&intTry)
	if err == nil {
		if intTry == 0 {
			*b = "false"
		} else {
			*b = "true"
		}
		return nil
	}

	// unable to unmarshal as bool, try string
	err = unmarshal(&stringTry)
	if err == nil {
		*b = QuotedBool(stringTry)
		return nil
	}

	return errors.Wrapf(err, "unable to unmarshal as bool, int or string")
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (QuotedBool) OpenAPISchemaType() []string { return []string{"string"} }

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (QuotedBool) OpenAPISchemaFormat() string { return "quoted-bool" }
