package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_vaidateMetricsData(t *testing.T) {
	tests := []struct {
		name    string
		data    ApplicationMetricsData
		wantErr bool
	}{
		{
			name: "all values are valid",
			data: ApplicationMetricsData{
				"key1": "val1",
				"key2": 6,
				"key3": 6.6,
				"key4": true,
			},
			wantErr: false,
		},
		{
			name:    "no data",
			data:    ApplicationMetricsData{},
			wantErr: true,
		},
		{
			name: "array value",
			data: ApplicationMetricsData{
				"key1": 10,
				"key2": []string{"val1", "val2"},
			},
			wantErr: true,
		},
		{
			name: "map value",
			data: ApplicationMetricsData{
				"key1": 10,
				"key2": map[string]string{"key1": "val1"},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := vaidateMetricsData(test.data)
			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

}
