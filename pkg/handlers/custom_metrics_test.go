package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	app := apptypes.App{
		ID: appID,
		License: fmt.Sprintf(`apiVersion: kots.io/v1beta1
kind: License
spec:
  licenseID: 2ULcK9BJd1dHGetHYZIysK9IADZ
  endpoint: %s`, server.URL),
	}

	mockStore := mock_store.NewMockStore(ctrl)
	mockStore.EXPECT().ListInstalledApps().Times(1).Return([]*apptypes.App{&app}, nil)

	handler := Handler{}
	clientWriter := httptest.NewRecorder()
	clientRequest := &http.Request{
		Body: io.NopCloser(bytes.NewBuffer(customMetricsData)),
	}

	// Validate

	handler.GetSendCustomApplicationMetricsHandler(mockStore)(clientWriter, clientRequest)

	req.Equal(http.StatusOK, clientWriter.Code)
}
