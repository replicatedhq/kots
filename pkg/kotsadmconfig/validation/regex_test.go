package validation

import (
	"reflect"
	"testing"

	configtypes "github.com/replicatedhq/kots/pkg/kotsadmconfig/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func Test_regexValidator_Validate(t *testing.T) {
	type fields struct {
		RegexValidator *kotsv1beta1.RegexValidator
	}
	type args struct {
		input string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *configtypes.ValidationError
		wantErr bool
	}{
		{
			name: "valid regex",
			fields: fields{
				RegexValidator: &kotsv1beta1.RegexValidator{
					Pattern: ".*",
				},
			},
			args: args{
				input: "test",
			},
			want: nil,
		}, {
			name: "invalid regex",
			fields: fields{
				RegexValidator: &kotsv1beta1.RegexValidator{
					Pattern: "[",
					Message: "must be a valid regex",
				},
			},
			args: args{
				input: "test",
			},
			want:    nil,
			wantErr: true,
		}, {
			name: "invalid input",
			fields: fields{
				RegexValidator: &kotsv1beta1.RegexValidator{
					Pattern: "test",
					Message: "must be a valid regex",
				},
			},
			args: args{
				input: "foo",
			},
			want: &configtypes.ValidationError{
				Message: "must be a valid regex",
			},
		}, {
			name: "invalid input with no message",
			fields: fields{
				RegexValidator: &kotsv1beta1.RegexValidator{
					Pattern: "test",
				},
			},
			args: args{
				input: "foo",
			},
			want: &configtypes.ValidationError{
				Message: regexMatchError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &regexValidator{
				RegexValidator: tt.fields.RegexValidator,
			}
			got, err := v.Validate(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("regexValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("regexValidator.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}
