package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			name: "array",
			values: map[string]MappedChartValue{
				"queues": MappedChartValue{
					valueType: "array",
					array: []map[string]*MappedChartValue{
						map[string]*MappedChartValue{
							"queue": &MappedChartValue{
								strValue:  "first",
								valueType: "string",
							},
							"replicas": &MappedChartValue{
								floatValue: float64(5),
								valueType:  "float",
							},
						},
					},
				},
			},
			expect: []string{
				"queues[0].queue=first",
				"queues[0].replicas=5",
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
							},
						},
					},
				},
			},
			expect: []string{"storage.postgres.enabled=true"},
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
							},
						},
					},
				},
			},
			expect: []string{
				"replicas=4",
				"storage.postgres.enabled=true",
				"storage.postgres.host=amazonaws.com",
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
							},
						},
					},
				},
			},
			expect: []string{
				"ingress.enabled=true",
				`ingress.annotations.kubernetes\.io/ingress\.class=nginx`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			h := HelmChartSpec{
				Values: test.values,
			}
			actual, err := h.RenderValues(h.Values)
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := test.mappedChartValue.GetValue()
			req.NoError(err)

			assert.Equal(t, test.expected, actual)
		})
	}
}
