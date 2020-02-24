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
			name: "chained configOptionValue",
			// the items further down the chain should include configOption values from their parents
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
						Default: "hello world replacedAbcItemValue",
					},
					"childItem2": {
						Value:   "",
						Default: "there should be something here this is a middle value hello world replacedAbcItemValue",
					},
					"childItem3": {
						Value:   "",
						Default: "this is a middle value hello world replacedAbcItemValue",
					},
				},
			},
		},
		{
			name: "readonly and provided values",
			// readonly configItems should update every time.
			// non-readonly items should update iff there is not a value provided.
			// values should override defaults when chaining template functions.
			args: args{
				configGroups: []kotsv1beta1.ConfigGroup{
					{
						Name:        "abc",
						Title:       "abc",
						Description: "abc",
						Items: []kotsv1beta1.ConfigItem{
							{ // a provided value overrides a default
								Name:  "abcItem",
								Type:  "text",
								Title: "abcItem",
								Default: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: `repl{{ ToUpper "hello, world"}}`,
								},
								Value: multitype.BoolOrString{},
							},
							{ // if no value is provided, the value is not overridden - and ConfigOption "blah" prefers the value over default if both exist
								Name: "childItem",
								Type: "text",
								Default: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: "the default value",
								},
								Value: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: `hello world repl{{ ConfigOption "abcItem" }}`,
								},
							},
							{ // despite having a value provided, this is readonly and so that value should not be provided
								Name:     "readonlyChild",
								Type:     "text",
								ReadOnly: true,
								Default: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: `chained value: repl{{ ConfigOption "childItem" }}`,
								},
								Value: multitype.BoolOrString{},
							},
							{ // this item's value should be overwritten by the provided value
								Name: "overwrittenChild",
								Type: "text",
								Value: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: `this is a middle value repl{{ ConfigOption "childItem" }}`,
								},
							},
						},
					},
				},
				templateContext: map[string]ItemValue{
					"abcItem": {
						Value: "no func",
					},
					"readonlyChild": {
						Value: "readonly provided value",
					},
					"overwrittenChild": {
						Value: "overwritten default",
					},
				},
				cipher: nil,
			},
			want: &ConfigCtx{
				ItemValues: map[string]ItemValue{
					"abcItem": {
						Value: "no func",
					},
					"childItem": {
						Default: "the default value",
						Value:   "hello world no func",
					},
					"readonlyChild": {
						Value:   "",
						Default: "chained value: hello world no func",
					},
					"overwrittenChild": {
						Value: "overwritten default",
					},
				},
			},
		},
		{
			name: "items not listed in config should remain untouched",
			args: args{
				configGroups: []kotsv1beta1.ConfigGroup{},
				templateContext: map[string]ItemValue{
					"item": {
						Value: "item does not exist",
					},
				},
				cipher: nil,
			},
			want: &ConfigCtx{ItemValues: map[string]ItemValue{
				"item": {
					Value: "item does not exist",
				},
			}},
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
