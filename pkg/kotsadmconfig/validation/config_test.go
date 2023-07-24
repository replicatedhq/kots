package validation

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/crypto"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kotskinds/multitype"
)

func Test_getValidatableItemValue(t *testing.T) {
	decryptedPassword := "password"
	encodedPassword := base64.StdEncoding.EncodeToString(crypto.Encrypt([]byte(decryptedPassword)))
	fileContent := "this is a file content"
	encodedFileContent := base64.StdEncoding.EncodeToString([]byte(fileContent))
	type args struct {
		value    multitype.BoolOrString
		itemType string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "empty type should default to text",
			args: args{
				value: multitype.BoolOrString{StrVal: "test"},
			},
			want: "test",
		}, {
			name: "string",
			args: args{
				value:    multitype.BoolOrString{StrVal: "test"},
				itemType: configtypes.TextItemType,
			},
			want: "test",
		}, {
			name: "textarea",
			args: args{
				value:    multitype.BoolOrString{StrVal: "test"},
				itemType: configtypes.TextAreaItemType,
			},
			want: "test",
		}, {
			name: "password",
			args: args{
				value:    multitype.BoolOrString{StrVal: encodedPassword},
				itemType: configtypes.PasswordItemType,
			},
			want: decryptedPassword,
		}, {
			name: "password plain text",
			args: args{
				value:    multitype.BoolOrString{StrVal: decryptedPassword},
				itemType: configtypes.PasswordItemType,
			},
			want: decryptedPassword,
		}, {
			name: "valid base64 file",
			args: args{
				value:    multitype.BoolOrString{StrVal: encodedFileContent},
				itemType: configtypes.FileItemType,
			},
			want: fileContent,
		}, {
			name: "invalid base64 file",
			args: args{
				value:    multitype.BoolOrString{StrVal: "dGhpcyBpcyBhIGZpbGUgY29udGVudAo"},
				itemType: configtypes.FileItemType,
			},
			wantErr: true,
		}, {
			name: configtypes.HeadingItemType,
			args: args{
				value:    multitype.BoolOrString{StrVal: "test"},
				itemType: configtypes.HeadingItemType,
			},
			wantErr: true,
		}, {
			name: "number",
			args: args{
				value:    multitype.BoolOrString{StrVal: "1"},
				itemType: "number",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getValidatableItemValue(tt.args.value, tt.args.itemType)
			if (err != nil) != tt.wantErr {
				t.Errorf("getValidatableItemValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getValidatableItemValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

var (
	validRegexConfigItem = kotsv1beta1.ConfigItem{
		Name:  "validRegexConfigItem",
		Type:  "text",
		Value: multitype.BoolOrString{StrVal: "test"},
		Validation: &kotsv1beta1.ConfigItemValidation{
			Regex: &kotsv1beta1.RegexValidator{
				Pattern: "^[a-z]+$",
				Message: "must be a valid regex",
			},
		},
	}
	regexMatchFailedConfigItem = kotsv1beta1.ConfigItem{
		Name:  "regexMatchFailedConfigItem",
		Type:  "text",
		Value: multitype.BoolOrString{StrVal: "A123"},
		Validation: &kotsv1beta1.ConfigItemValidation{
			Regex: &kotsv1beta1.RegexValidator{
				Pattern: "^[a-z]+$",
				Message: "must be a valid regex",
			},
		},
	}
	requiredFalseValueEmptyConfigItem = kotsv1beta1.ConfigItem{
		Name:     "requiredFalseValueEmptyConfigItem",
		Type:     "text",
		Value:    multitype.BoolOrString{StrVal: ""},
		Required: false,
		Validation: &kotsv1beta1.ConfigItemValidation{
			Regex: &kotsv1beta1.RegexValidator{
				Pattern: "^[a-z]+$",
				Message: "must be a valid regex",
			},
		},
	}
	requiredTrueValueEmptyConfigItem = kotsv1beta1.ConfigItem{
		Name:     "requiredFalseValueEmptyConfigItem",
		Type:     "text",
		Value:    multitype.BoolOrString{StrVal: ""},
		Required: true,
		Validation: &kotsv1beta1.ConfigItemValidation{
			Regex: &kotsv1beta1.RegexValidator{
				Pattern: "^[a-z]+$",
				Message: "must be a valid regex",
			},
		},
	}
	regexMatchFailedRequiredConfigItem = kotsv1beta1.ConfigItem{
		Name:  "regexMatchFailedConfigItem",
		Type:  "text",
		Value: multitype.BoolOrString{StrVal: "A123"},
		Validation: &kotsv1beta1.ConfigItemValidation{
			Regex: &kotsv1beta1.RegexValidator{
				Pattern: "^[a-z]+$",
				Message: "must be a valid regex",
			},
		},
	}
	invalidRegexPatternConfigItem = kotsv1beta1.ConfigItem{
		Name:  "invalidRegexConfigItem",
		Type:  "text",
		Value: multitype.BoolOrString{StrVal: "123"},
		Validation: &kotsv1beta1.ConfigItemValidation{
			Regex: &kotsv1beta1.RegexValidator{
				Pattern: "^([a-z]+$",
				Message: "must be a valid regex",
			},
		},
	}
	noValidationConfigItem = kotsv1beta1.ConfigItem{
		Name:  "nonValidatedConfigItem",
		Type:  "text",
		Value: multitype.BoolOrString{StrVal: "test"},
	}
	invalidValueConfigItem = kotsv1beta1.ConfigItem{
		Name:  "invalidConfigItemValue",
		Type:  configtypes.FileItemType,
		Value: multitype.BoolOrString{StrVal: "dGhpcyBpcyBhIGZpbGUgY29udGVudAo"},
		Validation: &kotsv1beta1.ConfigItemValidation{
			Regex: &kotsv1beta1.RegexValidator{
				Pattern: "^[a-z]+$",
				Message: "must be a valid regex",
			},
		},
	}
)

func Test_validateConfigItem(t *testing.T) {
	type args struct {
		item kotsv1beta1.ConfigItem
	}
	tests := []struct {
		name    string
		args    args
		want    *configtypes.ConfigItemValidationError
		wantErr bool
	}{
		{
			name: "valid regex",
			args: args{
				item: validRegexConfigItem,
			},
			want: nil,
		}, {
			name: "invalid regex pattern",
			args: args{
				item: invalidRegexPatternConfigItem,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid value config item",
			args: args{
				item: invalidValueConfigItem,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "non validatable config item",
			args: args{
				item: noValidationConfigItem,
			},
			want: nil,
		}, {
			name: "regex match failed",
			args: args{
				item: regexMatchFailedConfigItem,
			},
			want: &configtypes.ConfigItemValidationError{
				Name: regexMatchFailedConfigItem.Name,
				Type: regexMatchFailedConfigItem.Type,
				ValidationErrors: []configtypes.ValidationError{
					{
						Message: "must be a valid regex",
					},
				},
			},
		}, {
			name: "expect no error when required is false and value is empty",
			args: args{
				item: requiredFalseValueEmptyConfigItem,
			},
			want: nil,
		}, {
			name: "expect no error when required is true and value is empty",
			args: args{
				item: requiredTrueValueEmptyConfigItem,
			},
			want: nil,
		}, {
			name: "regex match failed required config item",
			args: args{
				item: regexMatchFailedRequiredConfigItem,
			},
			want: &configtypes.ConfigItemValidationError{
				Name: regexMatchFailedRequiredConfigItem.Name,
				Type: regexMatchFailedRequiredConfigItem.Type,
				ValidationErrors: []configtypes.ValidationError{
					{
						Message: "must be a valid regex",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateConfigItem(tt.args.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfigItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateConfigItem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateConfigItems(t *testing.T) {
	type args struct {
		configItems []kotsv1beta1.ConfigItem
	}
	tests := []struct {
		name    string
		args    args
		want    []configtypes.ConfigItemValidationError
		wantErr bool
	}{
		{
			name: "valid config items",
			args: args{
				configItems: []kotsv1beta1.ConfigItem{
					validRegexConfigItem,
					noValidationConfigItem,
				},
			},
			want:    []configtypes.ConfigItemValidationError{},
			wantErr: false,
		},
		{
			name: "invalid config validation regex pattern",
			args: args{
				configItems: []kotsv1beta1.ConfigItem{
					validRegexConfigItem,
					invalidRegexPatternConfigItem,
					noValidationConfigItem,
				},
			},
			want:    []configtypes.ConfigItemValidationError{},
			wantErr: true,
		},
		{
			name: "invalid config item values",
			args: args{
				configItems: []kotsv1beta1.ConfigItem{
					validRegexConfigItem,
					regexMatchFailedConfigItem,
					noValidationConfigItem,
				},
			},
			want: []configtypes.ConfigItemValidationError{
				{
					Name: "regexMatchFailedConfigItem",
					Type: "text",
					ValidationErrors: []configtypes.ValidationError{
						{Message: "must be a valid regex"},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateConfigItems(tt.args.configItems)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfigItems() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(len(got), len(tt.want)) {
				t.Errorf("len(validateConfigItems()) = %v, want %v", len(got), len(tt.want))
				t.Errorf("validateConfigItems() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if !reflect.DeepEqual(got[i].Name, tt.want[i].Name) ||
					!reflect.DeepEqual(got[i].Type, tt.want[i].Type) ||
					!reflect.DeepEqual(got[i].ValidationErrors, tt.want[i].ValidationErrors) {
					t.Errorf("validateConfigItems() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func Test_validateConfigGroup(t *testing.T) {
	type args struct {
		configGroup kotsv1beta1.ConfigGroup
	}
	tests := []struct {
		name    string
		args    args
		want    *configtypes.ConfigGroupValidationError
		wantErr bool
	}{
		{
			name: "invalid regex pattern config group",
			args: args{
				configGroup: kotsv1beta1.ConfigGroup{
					Name: "test",
					Items: []kotsv1beta1.ConfigItem{
						validRegexConfigItem,
						noValidationConfigItem,
						invalidRegexPatternConfigItem,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid config group",
			args: args{
				configGroup: kotsv1beta1.ConfigGroup{
					Name: "test",
					Items: []kotsv1beta1.ConfigItem{
						validRegexConfigItem,
						noValidationConfigItem,
					},
				},
			},
			want: nil,
		}, {
			name: "invalid config group",
			args: args{
				configGroup: kotsv1beta1.ConfigGroup{
					Name: "test",
					Items: []kotsv1beta1.ConfigItem{
						validRegexConfigItem,
						regexMatchFailedConfigItem,
						noValidationConfigItem,
					},
				},
			},
			want: &configtypes.ConfigGroupValidationError{
				Name: "test",
				ItemErrors: []configtypes.ConfigItemValidationError{
					{
						Name: regexMatchFailedConfigItem.Name,
						Type: regexMatchFailedConfigItem.Type,
						ValidationErrors: []configtypes.ValidationError{
							{
								Message: regexMatchFailedConfigItem.Validation.Regex.Message,
							},
						},
					},
				},
			},
		}, {
			name: "expect no error for empty config group",
			args: args{
				configGroup: kotsv1beta1.ConfigGroup{
					Name:  "test",
					Items: []kotsv1beta1.ConfigItem{},
				},
			},
			want: nil,
		}, {
			name: "expect no error for nil config group",
			args: args{
				configGroup: kotsv1beta1.ConfigGroup{
					Name: "test",
				},
			},
			want: nil,
		}, {
			name: "expect no error for group with items and group.when is false",
			args: args{
				configGroup: kotsv1beta1.ConfigGroup{
					Name: "test",
					When: "false",
					Items: []kotsv1beta1.ConfigItem{
						validRegexConfigItem,
						regexMatchFailedConfigItem,
						noValidationConfigItem,
					},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateConfigGroup(tt.args.configGroup)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfigGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateConfigGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateConfigSpec(t *testing.T) {
	type args struct {
		configSpec kotsv1beta1.ConfigSpec
	}
	tests := []struct {
		name    string
		args    args
		want    []configtypes.ConfigGroupValidationError
		wantErr bool
	}{
		{
			name: "valid config spec",
			args: args{
				configSpec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name: "test",
							Items: []kotsv1beta1.ConfigItem{
								validRegexConfigItem,
								noValidationConfigItem,
							},
						},
					},
				},
			},
			want: nil,
		}, {
			name: "invalid regex pattern config spec",
			args: args{
				configSpec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name: "test",
							Items: []kotsv1beta1.ConfigItem{
								validRegexConfigItem,
								noValidationConfigItem,
								invalidRegexPatternConfigItem,
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		}, {
			name: "valid config spec with non validatable config item",
			args: args{
				configSpec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name: "test",
							Items: []kotsv1beta1.ConfigItem{
								noValidationConfigItem,
							},
						},
					},
				},
			},
			want: nil,
		}, {
			name: "invalid config spec",
			args: args{
				configSpec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name: "test",
							Items: []kotsv1beta1.ConfigItem{
								validRegexConfigItem,
								regexMatchFailedConfigItem,
								noValidationConfigItem,
							},
						},
					},
				},
			},
			want: []configtypes.ConfigGroupValidationError{
				{
					Name: "test",
					ItemErrors: []configtypes.ConfigItemValidationError{
						{
							Name: regexMatchFailedConfigItem.Name,
							Type: regexMatchFailedConfigItem.Type,
							ValidationErrors: []configtypes.ValidationError{
								{
									Message: regexMatchFailedConfigItem.Validation.Regex.Message,
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateConfigSpec(tt.args.configSpec)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfigSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateConfigSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}
