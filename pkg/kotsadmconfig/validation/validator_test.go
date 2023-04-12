package validation

import (
	"reflect"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

func Test_isValidatableConfigItem(t *testing.T) {
	validValidator := &kotsv1beta1.ConfigItemValidation{
		Regex: &kotsv1beta1.RegexValidator{
			Pattern: ".*",
		},
	}
	tests := []struct {
		name string
		item kotsv1beta1.ConfigItem
		want bool
	}{
		{
			name: "valid",
			item: kotsv1beta1.ConfigItem{Type: "text", Validation: validValidator},
			want: true,
		}, {
			name: "invalid type",
			item: kotsv1beta1.ConfigItem{Type: "bool", Validation: validValidator},
		}, {
			name: "nil validation",
			item: kotsv1beta1.ConfigItem{Type: "text"},
		}, {
			name: "hidden",
			item: kotsv1beta1.ConfigItem{Type: "text", Validation: validValidator, Hidden: true},
		}, {
			name: "when false",
			item: kotsv1beta1.ConfigItem{Type: "text", Validation: validValidator, When: "false"},
		}, {
			name: "repeatable",
			item: kotsv1beta1.ConfigItem{Type: "text", Validation: validValidator, Repeatable: true},
		}, {
			name: "label",
			item: kotsv1beta1.ConfigItem{Type: "label", Validation: validValidator},
		}, {
			name: "heading",
			item: kotsv1beta1.ConfigItem{Type: "heading", Validation: validValidator},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidatableConfigItem(tt.item); got != tt.want {
				t.Errorf("isValidatableConfigItem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateRegex(t *testing.T) {
	type args struct {
		value          string
		regexValidator *kotsv1beta1.RegexValidator
	}
	tests := []struct {
		name string
		args args
		want *ValidationError
	}{
		{
			name: "valid",
			args: args{
				value: "foo",
				regexValidator: &kotsv1beta1.RegexValidator{
					Pattern: ".*",
				},
			},
			want: nil,
		}, {
			name: "invalid",
			args: args{
				value:          "foo",
				regexValidator: &kotsv1beta1.RegexValidator{Pattern: "bar"},
			},
			want: &ValidationError{
				ValidationErrorMessage: "Value does not match regex",
				RegexValidator:         &kotsv1beta1.RegexValidator{Pattern: "bar"},
			},
		}, {
			name: "invalid regex",
			args: args{
				value:          "foo",
				regexValidator: &kotsv1beta1.RegexValidator{Pattern: "["},
			},
			want: &ValidationError{
				ValidationErrorMessage: "Invalid regex: error parsing regexp: missing closing ]: `[`",
				RegexValidator: &kotsv1beta1.RegexValidator{
					Pattern: "[",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateRegex(tt.args.value, tt.args.regexValidator); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateRegex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validate(t *testing.T) {
	type args struct {
		value     string
		validator kotsv1beta1.ConfigItemValidation
	}
	tests := []struct {
		name string
		args args
		want []ValidationError
	}{
		{
			name: "valid regex",
			args: args{
				value: "foo",
				validator: kotsv1beta1.ConfigItemValidation{
					Regex: &kotsv1beta1.RegexValidator{Pattern: ".*"},
				},
			},
			want: nil,
		}, {
			name: "invalid regex",
			args: args{
				value: "foo",
				validator: kotsv1beta1.ConfigItemValidation{
					Regex: &kotsv1beta1.RegexValidator{Pattern: "["},
				},
			},
			want: []ValidationError{
				{
					ValidationErrorMessage: "Invalid regex: error parsing regexp: missing closing ]: `[`",
					RegexValidator:         &kotsv1beta1.RegexValidator{Pattern: "["},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validate(tt.args.value, tt.args.validator); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
