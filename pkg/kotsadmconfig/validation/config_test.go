package validation

import (
	"encoding/base64"
	"reflect"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	"github.com/replicatedhq/kots/pkg/crypto"
	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
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
			},
		},
	}
	invalidRegexConfigItem = kotsv1beta1.ConfigItem{
		Name:  "invalidRegexConfigItem",
		Type:  "text",
		Value: multitype.BoolOrString{StrVal: "123"},
		Validation: &kotsv1beta1.ConfigItemValidation{
			Regex: &kotsv1beta1.RegexValidator{
				Pattern: "^[a-z]+$",
			},
		},
	}
	nonValidatableConfigItem = kotsv1beta1.ConfigItem{
		Name:  "nonValidatedConfigItem",
		Type:  "text",
		Value: multitype.BoolOrString{StrVal: "test"},
	}
	invalidConfigItemValue = kotsv1beta1.ConfigItem{
		Name:  "invalidConfigItemValue",
		Type:  configtypes.FileItemType,
		Value: multitype.BoolOrString{StrVal: "dGhpcyBpcyBhIGZpbGUgY29udGVudAo"},
		Validation: &kotsv1beta1.ConfigItemValidation{
			Regex: &kotsv1beta1.RegexValidator{
				Pattern: "^[a-z]+$",
			},
		},
	}
)

func Test_validateConfigItem(t *testing.T) {
	type args struct {
		item kotsv1beta1.ConfigItem
	}
	tests := []struct {
		name string
		args args
		want *configtypes.ConfigItemValidationError
	}{
		{
			name: "valid regex",
			args: args{
				item: validRegexConfigItem,
			},
			want: nil,
		}, {
			name: "invalid regex",
			args: args{
				item: invalidRegexConfigItem,
			},
			want: &configtypes.ConfigItemValidationError{
				Name:  invalidRegexConfigItem.Name,
				Type:  invalidRegexConfigItem.Type,
				Value: invalidRegexConfigItem.Value,
				ValidationErrors: []configtypes.ValidationError{
					{
						ValidationErrorMessage: regexMatchError,
						RegexValidator:         invalidRegexConfigItem.Validation.Regex,
					},
				},
			},
		},
		{
			name: "invalid config item value",
			args: args{
				item: invalidConfigItemValue,
			},
			want: &configtypes.ConfigItemValidationError{
				Name:  invalidConfigItemValue.Name,
				Type:  invalidConfigItemValue.Type,
				Value: invalidConfigItemValue.Value,
				ValidationErrors: []configtypes.ValidationError{
					{
						ValidationErrorMessage: "failed to get item value: failed to base64 decode file item value: failed to bse64 decode interface data: illegal base64 data at input byte 28",
					},
				},
			},
		},
		{
			name: "non validatable config item",
			args: args{
				item: nonValidatableConfigItem,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateConfigItem(tt.args.item); !reflect.DeepEqual(got, tt.want) {
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
		name string
		args args
		want []configtypes.ConfigItemValidationError
	}{
		{
			name: "valid config items",
			args: args{
				configItems: []kotsv1beta1.ConfigItem{
					validRegexConfigItem,
					nonValidatableConfigItem,
				},
			},
			want: nil,
		}, {
			name: "invalid config items",
			args: args{
				configItems: []kotsv1beta1.ConfigItem{
					validRegexConfigItem,
					invalidRegexConfigItem,
					nonValidatableConfigItem,
				},
			},
			want: []configtypes.ConfigItemValidationError{
				{
					Name:  invalidRegexConfigItem.Name,
					Type:  invalidRegexConfigItem.Type,
					Value: invalidRegexConfigItem.Value,
					ValidationErrors: []configtypes.ValidationError{
						{
							ValidationErrorMessage: regexMatchError,
							RegexValidator:         invalidRegexConfigItem.Validation.Regex,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateConfigItems(tt.args.configItems); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateConfigItems() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateConfigGroup(t *testing.T) {
	type args struct {
		configGroup kotsv1beta1.ConfigGroup
	}
	tests := []struct {
		name string
		args args
		want *configtypes.ConfigGroupValidationError
	}{
		{
			name: "valid config group",
			args: args{
				configGroup: kotsv1beta1.ConfigGroup{
					Name: "test",
					Items: []kotsv1beta1.ConfigItem{
						validRegexConfigItem,
						nonValidatableConfigItem,
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
						invalidRegexConfigItem,
						nonValidatableConfigItem,
					},
				},
			},
			want: &configtypes.ConfigGroupValidationError{
				Name: "test",
				ItemErrors: []configtypes.ConfigItemValidationError{
					{
						Name:  invalidRegexConfigItem.Name,
						Type:  invalidRegexConfigItem.Type,
						Value: invalidRegexConfigItem.Value,
						ValidationErrors: []configtypes.ValidationError{
							{
								ValidationErrorMessage: regexMatchError,
								RegexValidator:         invalidRegexConfigItem.Validation.Regex,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateConfigGroup(tt.args.configGroup); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateConfigGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_hasConfigItemValidators(t *testing.T) {
	type args struct {
		configSpec kotsv1beta1.ConfigSpec
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "has validators",
			args: args{
				configSpec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Items: []kotsv1beta1.ConfigItem{
								validRegexConfigItem,
								nonValidatableConfigItem,
								invalidRegexConfigItem,
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "no validators",
			args: args{
				configSpec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Items: []kotsv1beta1.ConfigItem{
								nonValidatableConfigItem,
							},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasConfigItemValidators(tt.args.configSpec); got != tt.want {
				t.Errorf("hasConfigItemValidators() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateConfigSpec(t *testing.T) {
	type args struct {
		configSpec kotsv1beta1.ConfigSpec
	}
	tests := []struct {
		name string
		args args
		want []configtypes.ConfigGroupValidationError
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
								nonValidatableConfigItem,
							},
						},
					},
				},
			},
			want: nil,
		}, {
			name: "valid config spec with non validatable config item",
			args: args{
				configSpec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Name: "test",
							Items: []kotsv1beta1.ConfigItem{
								nonValidatableConfigItem,
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
								invalidRegexConfigItem,
								nonValidatableConfigItem,
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
							Name:  invalidRegexConfigItem.Name,
							Type:  invalidRegexConfigItem.Type,
							Value: invalidRegexConfigItem.Value,
							ValidationErrors: []configtypes.ValidationError{
								{
									ValidationErrorMessage: regexMatchError,
									RegexValidator:         invalidRegexConfigItem.Validation.Regex,
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
			if got := ValidateConfigSpec(tt.args.configSpec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateConfigSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}
