package template

import (
	"encoding/base64"
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuilder_NewConfigContext(t *testing.T) {
	type args struct {
		configGroups    []kotsv1beta1.ConfigGroup
		templateContext map[string]ItemValue
		cipher          *crypto.AESCipher
		license         *kotsv1beta1.License
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
		{
			name: "chain from license template func",
			args: args{
				configGroups: []kotsv1beta1.ConfigGroup{
					{
						Name:        "abc",
						Title:       "abc",
						Description: "abc",
						Items: []kotsv1beta1.ConfigItem{
							{
								Name:    "abcItem",
								Type:    "text",
								Title:   "abcItem",
								Default: multitype.BoolOrString{},
								Value: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: `license val: repl{{ LicenseFieldValue "abcField" }}`},
							},
						},
					},
					{
						Name:        "chain",
						Title:       "chain",
						Description: "chain",
						Items: []kotsv1beta1.ConfigItem{
							{
								Name:    "chainItem",
								Type:    "text",
								Title:   "chainItem",
								Default: multitype.BoolOrString{},
								Value: multitype.BoolOrString{
									Type:   multitype.String,
									StrVal: `chain val: repl{{ ConfigOption "abcItem" }}`},
							},
						},
					},
				},
				templateContext: map[string]ItemValue{},
				cipher:          nil,
				license: &kotsv1beta1.License{
					Spec: kotsv1beta1.LicenseSpec{
						Entitlements: map[string]kotsv1beta1.EntitlementField{
							"abcField": kotsv1beta1.EntitlementField{
								Value: kotsv1beta1.EntitlementValue{
									Type:   kotsv1beta1.String,
									StrVal: "abcValue",
								},
							},
						},
					},
				},
			},
			want: &ConfigCtx{
				ItemValues: map[string]ItemValue{
					"abcItem": {
						Value:   "license val: abcValue",
						Default: "",
					},
					"chainItem": {
						Value:   "chain val: license val: abcValue",
						Default: "",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			// expect license to be the one passed as an arg unless the test overrides this
			if tt.want.license == nil && tt.args.license != nil {
				tt.want.license = tt.args.license
			}

			builder := Builder{}
			builder.AddCtx(StaticCtx{})

			localRegistry := LocalRegistry{}
			got, err := builder.newConfigContext(tt.args.configGroups, tt.args.templateContext, localRegistry, tt.args.cipher, tt.args.license, nil)
			req.NoError(err)
			req.Equal(tt.want, got)
		})
	}
}

func Test_localImageName(t *testing.T) {
	ctxWithRegistry := ConfigCtx{
		LocalRegistry: LocalRegistry{
			Host:      "my.registry.com",
			Namespace: "my_namespace",
			Username:  "my_user",
			Password:  "my_password",
		},

		license: &kotsv1beta1.License{
			Spec: kotsv1beta1.LicenseSpec{
				Endpoint: "replicated.registry.com",
			},
		},
		app: &kotsv1beta1.Application{
			Spec: kotsv1beta1.ApplicationSpec{
				ProxyPublicImages: false,
			},
		},
	}

	ctxWithoutRegistry := ConfigCtx{
		LocalRegistry: LocalRegistry{},

		license: &kotsv1beta1.License{
			Spec: kotsv1beta1.LicenseSpec{
				AppSlug:  "myslug",
				Endpoint: "replicated.registry.com",
			},
		},
		app: &kotsv1beta1.Application{
			Spec: kotsv1beta1.ApplicationSpec{
				ProxyPublicImages: false,
			},
		},
	}

	ctxWithoutRegistryProxyAll := ConfigCtx{
		LocalRegistry: LocalRegistry{},

		license: &kotsv1beta1.License{
			Spec: kotsv1beta1.LicenseSpec{
				AppSlug:  "myslug",
				Endpoint: "replicated.registry.com",
			},
		},
		app: &kotsv1beta1.Application{
			Spec: kotsv1beta1.ApplicationSpec{
				ProxyPublicImages: true,
			},
		},
	}

	ctxWithNothing := ConfigCtx{
		LocalRegistry: LocalRegistry{},
	}

	tests := []struct {
		name     string
		ctx      ConfigCtx
		image    string
		expected string
	}{
		{
			name:     "rewrite public image to local",
			ctx:      ctxWithRegistry,
			image:    "nginx:latest",
			expected: "my.registry.com/my_namespace/nginx:latest",
		},
		{
			name:     "rewrite private image to local",
			ctx:      ctxWithRegistry,
			image:    "registry.replicated.com/kots/myimage:abcd123",
			expected: "my.registry.com/my_namespace/myimage:abcd123",
		},
		{
			name:     "do not rewrite public image",
			ctx:      ctxWithoutRegistry,
			image:    "redis:latest",
			expected: "redis:latest",
		},
		{
			name:     "rewrite public image when ProxyPublicImages: true",
			ctx:      ctxWithoutRegistryProxyAll,
			image:    "redis:latest",
			expected: "proxy.replicated.com/proxy/myslug/redis:latest",
		},
		{
			name:     "rewrite private image to proxy",
			ctx:      ctxWithoutRegistry,
			image:    "quay.io/replicated/myimage@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
			expected: "proxy.replicated.com/proxy/myslug/quay.io/replicated/myimage@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
		},
		{
			name:     "do not rewrite private replicated image to proxy",
			ctx:      ctxWithoutRegistry,
			image:    "registry.replicated.com/kots/myimage:v1.13.0",
			expected: "registry.replicated.com/kots/myimage:v1.13.0",
		},
		{
			name:     "do not panic when no license or registry are provided",
			ctx:      ctxWithNothing,
			image:    "quay.io/replicated/myimage@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
			expected: "proxy.replicated.com/proxy//quay.io/replicated/myimage@sha256:45b23dee08af5e43a7fea6c4cf9c25ccf269ee113168c19722f87876677c5cb2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			newName := test.ctx.localImageName(test.image)
			req.Equal(test.expected, newName)
		})
	}
}

func TestConfigCtx_localRegistryImagePullSecret(t *testing.T) {
	tests := []struct {
		name          string
		LocalRegistry LocalRegistry
		license       *kotsv1beta1.License
		want          string
	}{
		{
			name: "nil license",
			LocalRegistry: LocalRegistry{
				Host:      "",
				Namespace: "",
				Username:  "",
				Password:  "",
			},
			license: nil,
			want:    `{"auths":{"proxy.replicated.com":{"auth":"Og=="},"registry.replicated.com":{"auth":"Og=="}}}`,
		},
		{
			name: "licenseid abc",
			LocalRegistry: LocalRegistry{
				Host:      "",
				Namespace: "",
				Username:  "",
				Password:  "",
			},
			license: &kotsv1beta1.License{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kotsv1beta1.LicenseSpec{
					LicenseID: "abc",
				},
				Status: kotsv1beta1.LicenseStatus{},
			},
			want: `{"auths":{"proxy.replicated.com":{"auth":"YWJjOmFiYw=="},"registry.replicated.com":{"auth":"YWJjOmFiYw=="}}}`,
		},
		{
			name: "localregistry set",
			LocalRegistry: LocalRegistry{
				Host:      "example.com:5000",
				Namespace: "",
				Username:  "user",
				Password:  "password",
			},
			want: `{"auths":{"example.com:5000":{"auth":"dXNlcjpwYXNzd29yZA=="}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			ctx := ConfigCtx{
				LocalRegistry: tt.LocalRegistry,
				license:       tt.license,
			}
			want := base64.StdEncoding.EncodeToString([]byte(tt.want))
			got := ctx.localRegistryImagePullSecret()
			req.Equal(want, got)
		})
	}
}

func TestConfigCtx_semvers(t *testing.T) {
	req := require.New(t)

	ctx := ConfigCtx{}

	b, err := ctx.semverNE("1.2", "1.1")
	req.NoError(err)
	req.Equal(true, b)

	b, err = ctx.semverNE("1.2", "1.2")
	req.NoError(err)
	req.Equal(false, b)

	b, err = ctx.semverEQ("1.2", "1.1")
	req.NoError(err)
	req.Equal(false, b)

	b, err = ctx.semverEQ("1.2", "1.2")
	req.NoError(err)
	req.Equal(true, b)

	b, err = ctx.semverGT("1.2", "1.1")
	req.NoError(err)
	req.Equal(true, b)

	b, err = ctx.semverGT("1.1", "1.2")
	req.NoError(err)
	req.Equal(false, b)

	b, err = ctx.semverGTE("1.2", "1.1")
	req.NoError(err)
	req.Equal(true, b)

	b, err = ctx.semverGTE("1.2", "1.2")
	req.NoError(err)
	req.Equal(true, b)

	b, err = ctx.semverGTE("1.1", "1.2")
	req.NoError(err)
	req.Equal(false, b)

	b, err = ctx.semverLT("1.1", "1.2")
	req.NoError(err)
	req.Equal(true, b)

	b, err = ctx.semverLT("1.2", "1.1")
	req.NoError(err)
	req.Equal(false, b)

	b, err = ctx.semverLTE("1.1", "1.2")
	req.NoError(err)
	req.Equal(true, b)

	b, err = ctx.semverLTE("1.2", "1.2")
	req.NoError(err)
	req.Equal(true, b)

	b, err = ctx.semverLTE("1.2", "1.1")
	req.NoError(err)
	req.Equal(false, b)
}
