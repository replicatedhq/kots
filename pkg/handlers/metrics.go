package handlers

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/store"
)

func (h *Handler) GetAppMetrics(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, "")
}

type SendCustomAppMetricsRequest struct {
	Data CustomAppMetricsData `json:"data"`
}

type CustomAppMetricsData map[string]interface{}

func (h *Handler) GetSendCustomAppMetricsHandler(kotsStore store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if kotsadm.IsAirgap() {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("This request cannot be satisfied in airgap mode"))
			return
		}

		apps, err := kotsStore.ListInstalledApps()
		if err != nil {
			logger.Error(errors.Wrap(err, "list installed apps"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(apps) != 1 {
			logger.Errorf("custom application metrics can be sent only if one app is installed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		app := apps[0]
		license, err := kotsutil.LoadLicenseFromBytes([]byte(app.License))
		if err != nil {
			logger.Error(errors.Wrap(err, "load license from bytes"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		request := SendCustomAppMetricsRequest{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			logger.Error(errors.Wrap(err, "decode request"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := validateCustomAppMetricsData(request.Data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		err = replicatedapp.SendCustomAppMetricsData(license, app, request.Data)
		if err != nil {
			logger.Error(errors.Wrap(err, "set application data"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		JSON(w, http.StatusOK, "")
	}
}

func validateCustomAppMetricsData(data CustomAppMetricsData) error {
	if len(data) == 0 {
		return errors.New("no data provided")
	}

	for key, val := range data {
		valType := reflect.TypeOf(val)
		if valType == nil {
			return errors.Errorf("%s value is nil, only scalar values are allowed", key)
		}

		switch valType.Kind() {
		case reflect.Slice:
			return errors.Errorf("%s value is an array, only scalar values are allowed", key)
		case reflect.Array:
			return errors.Errorf("%s value is an array, only scalar values are allowed", key)
		case reflect.Map:
			return errors.Errorf("%s value is a map, only scalar values are allowed", key)
		}
	}

	return nil
}
