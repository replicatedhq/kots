// Based on https://github.com/kubernetes/apimachinery/blob/455a99f/pkg/util/intstr/intstr.go

package multitype

import (
	"encoding/json"
	"fmt"
	"strconv"

	fuzz "github.com/google/gofuzz"
	"github.com/pkg/errors"
)

// BoolOrString is a type that can hold an bool or a string.  When used in
// JSON or YAML marshalling and unmarshalling, it produces or consumes the
// inner type.  This allows you to have, for example, a JSON field that can
// accept a booolean string or raw bool.
//
// +protobuf=true
// +protobuf.options.(gogoproto.goproto_stringer)=false
// +k8s:openapi-gen=true
// +kubebuilder:validation:Type=BoolString
type BoolOrString struct {
	Type    BoolOrStringType `protobuf:"varbool,1,opt,name=type,casttype=Type" json:"-"`
	BoolVal bool             `protobuf:"varbool,2,opt,name=boolVal" json:"-"`
	StrVal  string           `protobuf:"bytes,3,opt,name=strVal" json:"-"`
}

// Type represents the stored type of BoolOrString.
type BoolOrStringType int

const (
	String BoolOrStringType = iota // The BoolOrString holds a string.
	Bool                           // The BoolOrString holds an bool.
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
// func Parse(val string) BoolOrString {
// 	// TODO: remove? this doesn't actually do any parsing
// 	return FromString(val)
// }

// Convert a string value into a BoolOrString with the same type as v
func (v BoolOrString) NewWithSameType(newValue string) (BoolOrString, error) {
	if v.Type == String {
		return FromString(newValue), nil
	}

	if newValue == "0" {
		return FromBool(false), nil
	}

	if newValue == "1" {
		return FromBool(true), nil
	}

	parsed, err := strconv.ParseBool(newValue)
	if err != nil {
		return BoolOrString{}, errors.Wrap(err, "failed to parse value")
	}

	return FromBool(parsed), nil
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

// String returns the string value, '1' for true, or '0' for false.
func (boolstr *BoolOrString) String() string {
	if boolstr.Type == String {
		return boolstr.StrVal
	} else if boolstr.BoolVal {
		return "1"
	} else {
		return "0"
	}
}

// Converts string to boolean if needed and returns its value
func (boolstr *BoolOrString) Boolean() (bool, error) {
	if boolstr.Type == Bool {
		return boolstr.BoolVal, nil
	}

	return strconv.ParseBool(boolstr.StrVal)
}

// Returns true if type is String and the value is the empty string
func (boolstr *BoolOrString) IsEmpty() bool {
	return boolstr.Type == String && boolstr.StrVal == ""
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

// BoolOrDefaultFalse returns bool val, if strValu is parsed returns parsed value  else false as default when parse error
func (boolstr *BoolOrString) BoolOrDefaultFalse() bool {
	val, err := boolstr.Bool()
	if err != nil {
		return false
	}
	return val
}

// Bool returns bool val, if strValu is parsed returns parsed value else false with parse error
func (boolstr *BoolOrString) Bool() (bool, error) {
	if boolstr.Type == Bool {
		return boolstr.BoolVal, nil
	}
	parsed, err := strconv.ParseBool(boolstr.StrVal)
	if err != nil {
		return false, fmt.Errorf("failed to parse bool string(err: %v)", err)
	}
	return parsed, nil
}
