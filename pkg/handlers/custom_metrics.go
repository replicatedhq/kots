package handlers

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/session"
)

type SendCustomApplicationMetricsRequest struct {
	Data ApplicationMetricsData `json:"data"`
}

type ApplicationMetricsData map[string]interface{}

func (h *Handler) SendCustomApplicationMetrics(w http.ResponseWriter, r *http.Request) {
	if kotsadm.IsAirgap() {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("This request cannot be satisfied in airgap mode"))
		return
	}

	license := session.ContextGetLicense(r)
	app := session.ContextGetApp(r)

	request := SendCustomApplicationMetricsRequest{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		logger.Error(errors.Wrap(err, "decode request"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := validateCustomMetricsData(request.Data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	err := replicatedapp.SendApplicationMetricsData(license, app, request.Data)
	if err != nil {
		logger.Error(errors.Wrap(err, "set application data"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	JSON(w, http.StatusOK, "")
}

func validateCustomMetricsData(data ApplicationMetricsData) error {
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
