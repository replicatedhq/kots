package ship

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_WatchNameFromState(t *testing.T) {
	tests := []struct {
		name         string
		stateJSON    string
		upstreamURI  string
		expectedName string
	}{
		{
			name:         "replicated.app",
			stateJSON:    shipEnterpriseStateJSON,
			expectedName: "Ship Enterprise Beta",
		},
		{
			name:         "consul helm chart",
			stateJSON:    consulHelmChartStateJSON,
			expectedName: "consul",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			watchName := WatchNameFromState([]byte(test.stateJSON))
			assert.Equal(t, test.expectedName, watchName)
		})
	}
}

func Test_TroubleshootCollectorsFromState(t *testing.T) {
	tests := []struct {
		name               string
		stateJSON          string
		expectedCollectors []byte
	}{
		{
			name:               "replicated.app",
			stateJSON:          shipEnterpriseStateJSON,
			expectedCollectors: []byte("collect:\n  v1:[]"),
		},
		{
			name:               "consul helm chart",
			stateJSON:          consulHelmChartStateJSON,
			expectedCollectors: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			collectors := TroubleshootCollectorsFromState([]byte(test.stateJSON))
			assert.Equal(t, test.expectedCollectors, collectors)
		})
	}
}

func Test_TroubleshootAnalyzersFromState(t *testing.T) {
	tests := []struct {
		name               string
		stateJSON          string
		expectedCollectors []byte
	}{
		{
			name:               "replicated.app",
			stateJSON:          shipEnterpriseStateJSON,
			expectedCollectors: []byte("analyze:\n  v1:[]"),
		},
		{
			name:               "consul helm chart",
			stateJSON:          consulHelmChartStateJSON,
			expectedCollectors: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			collectors := TroubleshootAnalyzersFromState([]byte(test.stateJSON))
			assert.Equal(t, test.expectedCollectors, collectors)
		})
	}
}
