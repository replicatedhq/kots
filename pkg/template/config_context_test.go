package template

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
)

func TestBuilder_NewConfigContext(t *testing.T) {
	type args struct {
		configGroups    []kotsv1beta1.ConfigGroup
		templateContext map[string]ItemValue
		cipher          *crypto.AESCipher
	}
	tests := []struct {
		name string
		args args
		want *ConfigCtx
	}{
		{
			name: "empty",
			args: args{
				configGroups:    []kotsv1beta1.ConfigGroup{},
				templateContext: map[string]ItemValue{},
				cipher:          nil,
			},
			want: &ConfigCtx{ItemValues: map[string]ItemValue{}},
		},
		{
			name: "configGroup",
			args: args{
				configGroups: []kotsv1beta1.ConfigGroup{
					{
						Name:        "abc",
						Title:       "abc",
						Description: "abc",
						Items: []kotsv1beta1.ConfigItem{
							{
								Name:  "abcItem",
								Type:  "text",
								Title: "abcItem",
								Default: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: "abcItemDefault",
								},
								Value: multitype.BoolOrString{},
							},
						},
					},
				},
				templateContext: map[string]ItemValue{},
				cipher:          nil,
			},
			want: &ConfigCtx{
				ItemValues: map[string]ItemValue{
					"abcItem": {
						Value:   "",
						Default: "abcItemDefault",
					},
				},
			},
		},
		{
			name: "configGroup and overriding template context value",
			args: args{
				configGroups: []kotsv1beta1.ConfigGroup{
					{
						Name:        "abc",
						Title:       "abc",
						Description: "abc",
						Items: []kotsv1beta1.ConfigItem{
							{
								Name:  "abcItem",
								Type:  "text",
								Title: "abcItem",
								Default: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: "abcItemDefault",
								},
								Value: multitype.BoolOrString{},
							},
						},
					},
				},
				templateContext: map[string]ItemValue{
					"abcItem": {
						Value: "replacedAbcItemValue",
					},
				},
				cipher: nil,
			},
			want: &ConfigCtx{
				ItemValues: map[string]ItemValue{
					"abcItem": {
						Value: "replacedAbcItemValue",
					},
				},
			},
		},
		{
			name: "chained configOptionValue failure", // this isn't really the desired behavior, test just demonstrates it
			args: args{
				configGroups: []kotsv1beta1.ConfigGroup{
					{
						Name:        "abc",
						Title:       "abc",
						Description: "abc",
						Items: []kotsv1beta1.ConfigItem{
							{
								Name:  "abcItem",
								Type:  "text",
								Title: "abcItem",
								Default: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: "abcItemDefault",
								},
								Value: multitype.BoolOrString{},
							},
							{
								Name: "childItem1",
								Type: "text",
								Default: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: `repl{{ ConfigOption "abcItem" }}`,
								},
								Value: multitype.BoolOrString{},
							},
							{
								Name: "childItem2",
								Type: "text",
								Default: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: `there should be something here repl{{ ConfigOption "childItem1" }}`,
								},
								Value: multitype.BoolOrString{},
							},
						},
					},
				},
				templateContext: map[string]ItemValue{
					"abcItem": {
						Value: "replacedAbcItemValue",
					},
				},
				cipher: nil,
			},
			want: &ConfigCtx{
				ItemValues: map[string]ItemValue{
					"abcItem": {
						Value: "replacedAbcItemValue",
					},
					"childItem1": {
						Value:   "",
						Default: "",
					},
					"childItem2": {
						Value:   "",
						Default: "",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()

			req := require.New(t)

			builder := Builder{}
			builder.AddCtx(StaticCtx{})

			got, err := builder.NewConfigContext(tt.args.configGroups, tt.args.templateContext, tt.args.cipher)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}
