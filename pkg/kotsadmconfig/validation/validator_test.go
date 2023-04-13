package validation

import (
	"reflect"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
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
			name: "valid empty type",
			item: kotsv1beta1.ConfigItem{Validation: validValidator},
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

func Test_validate(t *testing.T) {
	type args struct {
		value     string
		validator kotsv1beta1.ConfigItemValidation
	}
	tests := []struct {
		name    string
		args    args
		want    []configtypes.ValidationError
		wantErr bool
	}{
		{
			name: "valid regex",
			args: args{
				value: "foo",
				validator: kotsv1beta1.ConfigItemValidation{
					Regex: &kotsv1beta1.RegexValidator{
						Pattern: ".*",
						BaseValidator: kotsv1beta1.BaseValidator{
							Message: "must be a valid regex",
						},
					},
				},
			},
			want: nil,
		}, {
			name: "invalid regex pattern",
			args: args{
				value: "foo",
				validator: kotsv1beta1.ConfigItemValidation{
					Regex: &kotsv1beta1.RegexValidator{
						Pattern: "[",
						BaseValidator: kotsv1beta1.BaseValidator{
							Message: "must be a valid regex",
						}},
				},
			},
			want:    nil,
			wantErr: true,
		}, {
			name: "invalid value for regex pattern",
			args: args{
				value: "foo",
				validator: kotsv1beta1.ConfigItemValidation{
					Regex: &kotsv1beta1.RegexValidator{
						Pattern: "^[A-Z]+$",
						BaseValidator: kotsv1beta1.BaseValidator{
							Message: "must be a valid regex",
						}},
				},
			},
			want: []configtypes.ValidationError{
				{
					Message: "must be a valid regex",
				},
			},
		}, {
			name: "empty item validators",
			args: args{
				value:     "foo",
				validator: kotsv1beta1.ConfigItemValidation{},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validate(tt.args.value, tt.args.validator)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildValidators(t *testing.T) {
	regexpValidator := &kotsv1beta1.RegexValidator{Pattern: ".*"}
	type args struct {
		itemValidator kotsv1beta1.ConfigItemValidation
	}
	tests := []struct {
		name string
		args args
		want []validator
	}{
		{
			name: "regex",
			args: args{
				itemValidator: kotsv1beta1.ConfigItemValidation{
					Regex: regexpValidator,
				},
			},
			want: []validator{
				&regexValidator{
					regexpValidator,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildValidators(tt.args.itemValidator); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildValidators() = %v, want %v", got, tt.want)
			}
		})
	}
}
