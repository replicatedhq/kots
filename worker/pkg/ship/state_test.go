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
