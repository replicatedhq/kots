// Based on https://github.com/kubernetes/apimachinery/blob/455a99f/pkg/util/intstr/intstr.go

package multitype

import (
	"encoding/json"
	"fmt"

	fuzz "github.com/google/gofuzz"
)

// BoolOrString is a type that can hold an bool or a string.  When used in
// JSON or YAML marshalling and unmarshalling, it produces or consumes the
// inner type.  This allows you to have, for example, a JSON field that can
// accept a booolean string or raw bool.
//
// +protobuf=true
// +protobuf.options.(gogoproto.goproto_stringer)=false
// +k8s:openapi-gen=true
type BoolOrString struct {
	Type    BoolOrStringType `protobuf:"varbool,1,opt,name=type,casttype=Type"`
	BoolVal bool             `protobuf:"varbool,2,opt,name=boolVal"`
	StrVal  string           `protobuf:"bytes,3,opt,name=strVal"`
}

// Type represents the stored type of BoolOrString.
type BoolOrStringType int

const (
	Bool   BoolOrStringType = iota // The BoolOrString holds an bool.
	String                         // The BoolOrString holds a string.
)

// FromBool creates an BoolOrString object with a bool value.
func FromBool(val bool) BoolOrString {
	return BoolOrString{Type: Bool, BoolVal: val}
}

// FromString creates an BoolOrString object with a string value.
func FromString(val string) BoolOrString {
	return BoolOrString{Type: String, StrVal: val}
}

// Parse the given string
func Parse(val string) BoolOrString {
	return FromString(val)
}

// UnmarshalJSON implements the json.Unmarshaller boolerface.
func (boolstr *BoolOrString) UnmarshalJSON(value []byte) error {
	if value[0] == '"' {
		boolstr.Type = String
		return json.Unmarshal(value, &boolstr.StrVal)
	}
	boolstr.Type = Bool
	return json.Unmarshal(value, &boolstr.BoolVal)
}

// String returns the string value, '1' for true, or '' for false.
func (boolstr *BoolOrString) String() string {
	if boolstr.Type == String {
		return boolstr.StrVal
	} else if boolstr.BoolVal {
		return "1"
	} else {
		return ""
	}
}

// MarshalJSON implements the json.Marshaller interface.
func (boolstr BoolOrString) MarshalJSON() ([]byte, error) {
	switch boolstr.Type {
	case Bool:
		return json.Marshal(boolstr.BoolVal)
	case String:
		return json.Marshal(boolstr.StrVal)
	default:
		return []byte{}, fmt.Errorf("impossible BoolOrString.Type")
	}
}

// MarshalYAML implements the yaml.Marshaller interface https://godoc.org/gopkg.in/yaml.v3#Marshaler
func (boolstr BoolOrString) MarshalYAML() (interface{}, error) {
	switch boolstr.Type {
	case Bool:
		return boolstr.BoolVal, nil
	case String:
		return boolstr.StrVal, nil
	default:
		return []byte{}, fmt.Errorf("impossible BoolOrString.Type")
	}
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (BoolOrString) OpenAPISchemaType() []string { return []string{"string"} }

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (BoolOrString) OpenAPISchemaFormat() string { return "bool-or-string" }

func (boolstr *BoolOrString) Fuzz(c fuzz.Continue) {
	if boolstr == nil {
		return
	}
	if c.RandBool() {
		boolstr.Type = Bool
		c.Fuzz(&boolstr.BoolVal)
		boolstr.StrVal = ""
	} else {
		boolstr.Type = String
		boolstr.BoolVal = false
		c.Fuzz(&boolstr.StrVal)
	}
}
