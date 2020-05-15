package upstream

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
)

var (
	encryptionKeyForTest = "nGXiCcP71wgFMQY0sdIGumF521NempOFqBCFKP0M+X4hlOwd"
)

func Test_EncryptConfigValues(t *testing.T) {
	tests := []struct {
		name             string
		config           *kotsv1beta1.Config
		configValues     *kotsv1beta1.ConfigValues
		installation     *kotsv1beta1.Installation
		expect           *kotsv1beta1.ConfigValues
		expectErr        bool
		expectErrMessage string
	}{
		{
			name: "no config",
			installation: &kotsv1beta1.Installation{
				Spec: kotsv1beta1.InstallationSpec{
					EncryptionKey: "abcdef",
				},
			},
			expect: nil,
		},
		{
			name: "not plain text values",
			installation: &kotsv1beta1.Installation{
				Spec: kotsv1beta1.InstallationSpec{
					EncryptionKey: encryptionKeyForTest,
				},
			},
			config: &kotsv1beta1.Config{
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Items: []kotsv1beta1.ConfigItem{
								{
									Type: "password",
									Name: "aname",
								},
							},
						},
					},
				},
			},
			configValues: &kotsv1beta1.ConfigValues{
				Spec: kotsv1beta1.ConfigValuesSpec{
					Values: map[string]kotsv1beta1.ConfigValue{
						"aname": {
							ValuePlaintext: "avalue",
						},
					},
				},
			},
			expect: &kotsv1beta1.ConfigValues{
				Spec: kotsv1beta1.ConfigValuesSpec{
					Values: map[string]kotsv1beta1.ConfigValue{
						"aname": {
							Value: "rCs3+4Ja79/aaOerFGCK6as0RRK9Yw==",
						},
					},
				},
			},
		},
		{
			name: "not plain text values when a value is also supplied",
			installation: &kotsv1beta1.Installation{
				Spec: kotsv1beta1.InstallationSpec{
					EncryptionKey: encryptionKeyForTest,
				},
			},
			config: &kotsv1beta1.Config{
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Items: []kotsv1beta1.ConfigItem{
								{
									Type: "password",
									Name: "aname",
								},
							},
						},
					},
				},
			},
			configValues: &kotsv1beta1.ConfigValues{
				Spec: kotsv1beta1.ConfigValuesSpec{
					Values: map[string]kotsv1beta1.ConfigValue{
						"aname": {
							ValuePlaintext: "avalue",
							Value:          "wrong value",
						},
					},
				},
			},
			expect: &kotsv1beta1.ConfigValues{
				Spec: kotsv1beta1.ConfigValuesSpec{
					Values: map[string]kotsv1beta1.ConfigValue{
						"aname": {
							Value: "rCs3+4Ja79/aaOerFGCK6as0RRK9Yw==",
						},
					},
				},
			},
		},
		{
			name: "not plain text values object type is not password",
			installation: &kotsv1beta1.Installation{
				Spec: kotsv1beta1.InstallationSpec{
					EncryptionKey: encryptionKeyForTest,
				},
			},
			config: &kotsv1beta1.Config{
				Spec: kotsv1beta1.ConfigSpec{
					Groups: []kotsv1beta1.ConfigGroup{
						{
							Items: []kotsv1beta1.ConfigItem{
								{
									Type: "text",
									Name: "aname",
								},
							},
						},
					},
				},
			},
			configValues: &kotsv1beta1.ConfigValues{
				Spec: kotsv1beta1.ConfigValuesSpec{
					Values: map[string]kotsv1beta1.ConfigValue{
						"aname": {
							ValuePlaintext: "avalue",
						},
					},
				},
			},
			expect: &kotsv1beta1.ConfigValues{
				Spec: kotsv1beta1.ConfigValuesSpec{
					Values: map[string]kotsv1beta1.ConfigValue{
						"aname": {
							Value: "",
						},
					},
				},
			},
			expectErr:        true,
			expectErrMessage: `Cannot encrypt item "aname" because item type was "text" (not password)`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()
			req := require.New(t)

			actual, err := EncryptConfigValues(test.config, test.configValues, test.installation)
			if test.expectErr {
				assert.EqualError(t, err, test.expectErrMessage)
				return
			}

			req.NoError(err)

			if test.expect == nil {
				assert.Nil(t, actual)
				return
			}

			assert.EqualValues(t, test.expect.Spec.Values, actual.Spec.Values)
		})
	}
}
