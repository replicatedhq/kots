package handlers

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/session"
)

type SetApplicationMetricsRequest struct {
	Data ApplicationMetricsData `json:"data"`
}

type ApplicationMetricsData map[string]interface{}

func (h *Handler) SetApplicationMetrics(w http.ResponseWriter, r *http.Request) {
	license := session.ContextGetLicense(r)
	app := session.ContextGetApp(r)

	setApplicationMetricsRequest := SetApplicationMetricsRequest{}
	if err := json.NewDecoder(r.Body).Decode(&setApplicationMetricsRequest); err != nil {
		logger.Error(errors.Wrap(err, "decode request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := vaidateMetricsData(setApplicationMetricsRequest.Data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	err := replicatedapp.SetApplicationMetricsData(license, app, setApplicationMetricsRequest.Data)
	if err != nil {
		logger.Error(errors.Wrap(err, "set application data"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	JSON(w, http.StatusOK, "")
}

func vaidateMetricsData(data ApplicationMetricsData) error {
	if len(data) == 0 {
		return errors.New("no data provided")
	}

	for key, val := range data {
		valType := reflect.TypeOf(val)
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
