package handlers

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/session"
	"github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

func Test_validateCustomMetricsData(t *testing.T) {
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
			err := validateCustomMetricsData(test.data)
			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_SendCustomApplicationMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	req := require.New(t)
	customMetricsData := []byte(`{"data":{"key1_string":"val1","key2_int":5,"key3_float":1.5,"key4_numeric_string":"1.6"}}`)
	appID := "app-id-123"

	// Mock server side

	serverRouter := mux.NewRouter()
	server := httptest.NewServer(serverRouter)
	defer server.Close()

	serverRouter.Methods("POST").Path("/application/custom-metrics").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		req.NoError(err)
		req.Equal(string(customMetricsData), string(body))
		req.Equal(appID, r.Header.Get("X-Replicated-InstanceID"))
		w.WriteHeader(http.StatusOK)
	})

	// Mock kotsadm side

	os.Setenv("USE_MOCK_REPORTING", "1")
	defer os.Unsetenv("USE_MOCK_REPORTING")

	handler := Handler{}
	clientWriter := httptest.NewRecorder()
	clientRequest := &http.Request{
		Body: io.NopCloser(bytes.NewBuffer(customMetricsData)),
	}

	clientRequest = session.ContextSetLicense(clientRequest, &v1beta1.License{
		Spec: v1beta1.LicenseSpec{
			Endpoint: server.URL,
		},
	})
	clientRequest = session.ContextSetApp(clientRequest, &apptypes.App{
		ID: appID,
	})

	// Validate

	handler.SendCustomApplicationMetrics(clientWriter, clientRequest)

	req.Equal(http.StatusOK, clientWriter.Code)
}
