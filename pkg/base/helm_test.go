package base

import (
	"testing"

	"github.com/ghodss/yaml"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
)

func Test_checkChartForVersion(t *testing.T) {
	v3Test := map[string]interface{}{
		"apiVersion": "v2",
		"name":       "testChart",
		"type":       "application",
		"version":    "v0.0.1",
		"appVersion": "v1.0.0",
	}

	v2Test := map[string]interface{}{
		"name":       "testChart",
		"type":       "application",
		"version":    "v2",
		"appVersion": "v2",
	}

	tests := []struct {
		name    string
		chart   map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			name:    "version 3 API",
			chart:   v3Test,
			want:    "v3",
			wantErr: false,
		},
		{
			name:    "version 2",
			chart:   v2Test,
			want:    "v2",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlContent, err := yaml.Marshal(tt.chart)
			if err != nil {
				t.Errorf("checkChartForVersion() error = %v", err)
			}
			chartFile := upstreamtypes.UpstreamFile{
				Content: yamlContent,
			}
			got, err := checkChartForVersion(&chartFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkChartForVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkChartForVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
