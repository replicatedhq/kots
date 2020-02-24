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
			// ideally, the items further down the chain would include configOption values from their parents
			// for reference, ship calculated this outside of the config.builder function, but it might be worthwhile to bring things inside here
			// ship code: https://github.com/replicatedhq/ship/blob/1c68827add9e81979e12ef2a041710b1ff7f47e5/pkg/lifecycle/render/config/resolve/api.go#L109-L193
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
									StrVal: `hello world repl{{ ConfigOption "abcItem" }}`,
								},
								Value: multitype.BoolOrString{},
							},
							{
								Name: "childItem2",
								Type: "text",
								Default: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: `there should be something here repl{{ ConfigOption "childItem3" }}`, // this test case refers to item #3 to ensure that we aren't just rendering top to bottom, we build a dependency graph
								},
								Value: multitype.BoolOrString{},
							},
							{
								Name: "childItem3",
								Type: "text",
								Default: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: `this is a middle value repl{{ ConfigOption "childItem1" }}`,
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
						Default: "", // this is the current value
						// Default: "hello world replacedAbcItemValue", // this is the desired value
					},
					"childItem2": {
						Value:   "",
						Default: "", // this is the current value
						// Default: "there should be something here this is a middle value hello world replacedAbcItemValue", // this is the desired value
					},
					"childItem3": {
						Value:   "",
						Default: "", // this is the current value
						// Default: "this is a middle value hello world replacedAbcItemValue", // this is the desired value
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

			localRegistry := LocalRegistry{}
			got, err := builder.NewConfigContext(tt.args.configGroups, tt.args.templateContext, localRegistry, tt.args.cipher)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}
