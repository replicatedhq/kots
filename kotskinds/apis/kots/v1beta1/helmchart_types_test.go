package v1beta1

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_UnmarshalValues(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		expect map[string]MappedChartValue
	}{
		{
			name: "simple",
			value: `{
  "apiVersion": "kots.io/v1beta1",
  "kind": "HelmChart",
  "metadata": {
    "name": "test"
  },
  "spec": {
    "values": {
      "k8s": "blue"
    }
  }
}`,
			expect: map[string]MappedChartValue{
				"k8s": MappedChartValue{
					strValue:  "blue",
					valueType: "string",
				},
			},
		},
		{
			name: "array",
			value: `{
  "apiVersion": "kots.io/v1beta1",
  "kind": "HelmChart",
  "metadata": {
    "name": "test"
  },
  "spec": {
    "values": {
      "l": [
        {
	  "a": "b"
	},
	{
	  "a": "c"
	}
      ]
    }
  }
}`,
			expect: map[string]MappedChartValue{
				"l": MappedChartValue{
					valueType: "array",
					array: []*MappedChartValue{
						{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"a": &MappedChartValue{
									valueType: "string",
									strValue:  "b",
								},
							},
						},
						{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"a": &MappedChartValue{
									valueType: "string",
									strValue:  "c",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual := HelmChart{}
			err := json.Unmarshal([]byte(test.value), &actual)
			req.NoError(err)

			assert.Equal(t, test.expect, actual.Spec.Values)
		})
	}
}
func Test_HelmChartSpecRenderValues(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]MappedChartValue
		expect []string
	}{
		{
			name: "simple",
			values: map[string]MappedChartValue{
				"a": MappedChartValue{
					strValue:  "b",
					valueType: "string",
				},
			},
			expect: []string{"a=b"},
		},
		{
			name: "string with comma",
			values: map[string]MappedChartValue{
				"a": MappedChartValue{
					strValue:  "b,c,d",
					valueType: "string",
				},
			},
			expect: []string{`a=b\,c\,d`},
		},
		{
			name: "string with comma in generated string",
			values: map[string]MappedChartValue{
				"a": MappedChartValue{
					strValue:  "replaceme",
					valueType: "string",
				},
			},
			expect: []string{`a=I\,was\,replaced`},
		},
		{

			name: "with-child",
			values: map[string]MappedChartValue{
				"postgres": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"enabled": &MappedChartValue{
							boolValue: true,
							valueType: "bool",
						},
					},
				},
			},
			expect: []string{"postgres.enabled=true"},
		},
		{
			name: "children with array",
			values: map[string]MappedChartValue{
				"worker": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"queues": &MappedChartValue{
							valueType: "array",
							array: []*MappedChartValue{
								{
									valueType: "children",
									children: map[string]*MappedChartValue{
										"queue": &MappedChartValue{
											strValue:  "first",
											valueType: "string",
										},
										"replicas": &MappedChartValue{
											floatValue: float64(1),
											valueType:  "float",
										},
									},
								},
								{
									valueType: "children",
									children: map[string]*MappedChartValue{
										"queue": &MappedChartValue{
											strValue:  "second",
											valueType: "string",
										},
										"replicas": &MappedChartValue{
											floatValue: float64(2),
											valueType:  "float",
										},
									},
								},
							},
						},
					},
				},
			},
			expect: []string{
				"worker.queues[0].queue=first",
				"worker.queues[0].replicas=1",
				"worker.queues[1].queue=second",
				"worker.queues[1].replicas=2",
			},
		},
		{
			name: "with-deep-children",
			values: map[string]MappedChartValue{
				"storage": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"postgres": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"enabled": &MappedChartValue{
									boolValue: true,
									valueType: "bool",
								},
								"replacementtest": &MappedChartValue{
									strValue:  "replaceme",
									valueType: `string`,
								},
							},
						},
					},
				},
			},
			expect: []string{
				"storage.postgres.enabled=true",
				`storage.postgres.replacementtest=I\,was\,replaced`,
			},
		},
		{
			name: "complex",
			values: map[string]MappedChartValue{
				"replicas": MappedChartValue{
					valueType:  "float",
					floatValue: float64(4),
				},
				"storage": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"postgres": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"enabled": &MappedChartValue{
									boolValue: true,
									valueType: "bool",
								},
								"host": &MappedChartValue{
									strValue:  "amazonaws.com",
									valueType: "string",
								},
								"replacementtest": &MappedChartValue{
									strValue:  "replaceme",
									valueType: `string`,
								},
							},
						},
					},
				},
			},
			expect: []string{
				"replicas=4",
				"storage.postgres.enabled=true",
				"storage.postgres.host=amazonaws.com",
				`storage.postgres.replacementtest=I\,was\,replaced`,
			},
		},
		{
			name: "with a map",
			values: map[string]MappedChartValue{
				"ingress": MappedChartValue{
					valueType: "children",
					children: map[string]*MappedChartValue{
						"enabled": &MappedChartValue{
							boolValue: true,
							valueType: "bool",
						},
						"annotations": &MappedChartValue{
							valueType: "children",
							children: map[string]*MappedChartValue{
								"kubernetes.io/ingress.class": &MappedChartValue{
									strValue:  "nginx",
									valueType: "string",
								},
								"replacementtest": &MappedChartValue{
									strValue:  "replaceme please",
									valueType: `string`,
								},
							},
						},
					},
				},
			},
			expect: []string{
				"ingress.enabled=true",
				`ingress.annotations.kubernetes\.io/ingress\.class=nginx`,
				`ingress.annotations.replacementtest=I\,was\,replaced please`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			h := HelmChartSpec{
				Values: test.values,
			}
			actual, err := h.RenderValues(h.Values, func(s2 string) (s string, err error) {
				return strings.NewReplacer(
					"replaceme", "I,was,replaced",
				).Replace(s2), nil
			})
			req.NoError(err)

			assert.ElementsMatch(t, test.expect, actual)
		})
	}
}

func Test_MappedChartValueGetValue(t *testing.T) {
	tests := []struct {
		name             string
		mappedChartValue MappedChartValue
		expected         interface{}
	}{
		{
			name: "string",
			mappedChartValue: MappedChartValue{
				strValue:  "abc",
				valueType: "string",
			},
			expected: "abc",
		},
		{
			name: "string with comma",
			mappedChartValue: MappedChartValue{
				strValue:  "abc,def,ghi",
				valueType: "string",
			},
			expected: `abc\,def\,ghi`,
		},
		{
			name: "bool",
			mappedChartValue: MappedChartValue{
				boolValue: true,
				valueType: "bool",
			},
			expected: true,
		},
		{
			name: "float",
			mappedChartValue: MappedChartValue{
				floatValue: 42,
				valueType:  "float",
			},
			expected: float64(42),
		},
		{
			name: "children",
			mappedChartValue: MappedChartValue{
				valueType: "children",
				children: map[string]*MappedChartValue{
					"child": &MappedChartValue{
						strValue:  "val",
						valueType: "string",
					},
				},
			},
			expected: map[string]interface{}{
				"child": "val",
			},
		},
		{
			name: "array",
			mappedChartValue: MappedChartValue{
				valueType: "array",
				array: []*MappedChartValue{
					&MappedChartValue{
						strValue:  "val1",
						valueType: "string",
					},
					&MappedChartValue{
						strValue:  "val2",
						valueType: "string",
					},
				},
			},
			expected: []interface{}{
				"val9",
				"val2",
			},
		},
		{
			name: "children with array",
			mappedChartValue: MappedChartValue{
				valueType: "children",
				children: map[string]*MappedChartValue{
					"child": &MappedChartValue{
						array: []*MappedChartValue{
							&MappedChartValue{
								strValue:  "val1",
								valueType: "string",
							},
							&MappedChartValue{
								strValue:  "val2",
								valueType: "string",
							},
						},
						valueType: "array",
					},
				},
			},
			expected: map[string]interface{}{
				"child": []interface{}{
					"val9",
					"val2",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := test.mappedChartValue.GetBuiltValue(func(s2 string) (s string, err error) {
				return strings.NewReplacer(
					"1", `9`,
				).Replace(s2), nil
			})
			req.NoError(err)

			assert.Equal(t, test.expected, actual)
		})
	}
}
