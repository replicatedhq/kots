package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/reporting"
	upgradepreflight "github.com/replicatedhq/kots/pkg/upgradeservice/preflight"
	"github.com/replicatedhq/kots/pkg/util"
)

type GetPreflightResultResponse struct {
	PreflightProgress string                         `json:"preflightProgress,omitempty"`
	PreflightResult   preflighttypes.PreflightResult `json:"preflightResult"`
}

func (h *Handler) StartPreflightChecks(w http.ResponseWriter, r *http.Request) {
	params := GetContextParams(r)
	appSlug := mux.Vars(r)["appSlug"]

	if params.AppSlug != appSlug {
		logger.Error(errors.Errorf("app slug in path %s does not match app slug in context %s", appSlug, params.AppSlug))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	appLicense, err := kotsutil.LoadLicenseFromBytes([]byte(params.AppLicense))
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to load license from bytes"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	app := &apptypes.App{
		ID:       params.AppID,
		Slug:     params.AppSlug,
		IsAirgap: params.AppIsAirgap,
		IsGitOps: params.AppIsGitOps,
	}

	localRegistry := registrytypes.RegistrySettings{
		Hostname:   params.RegistryEndpoint,
		Username:   params.RegistryUsername,
		Password:   params.RegistryPassword,
		Namespace:  params.RegistryNamespace,
		IsReadOnly: params.RegistryIsReadOnly,
	}

	if err := upgradepreflight.ResetPreflightData(); err != nil {
		logger.Error(errors.Wrap(err, "failed to reset preflight data"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	reportingFn := func() error {
		if params.AppIsAirgap {
			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}
			report := reporting.BuildInstanceReport(appLicense.Spec.LicenseID, params.ReportingInfo)
			return reporting.AppendReport(clientset, util.PodNamespace, app.Slug, report)
		}
		return reporting.SendOnlineAppInfo(appLicense, params.ReportingInfo)
	}

	go func() {
		if err := upgradepreflight.Run(app, params.BaseArchive, params.NextSequence, localRegistry, false, reportingFn); err != nil {
			logger.Error(errors.Wrap(err, "failed to run preflights"))
			return
		}
	}()

	JSON(w, http.StatusOK, struct{}{})
}

func (h *Handler) GetPreflightResult(w http.ResponseWriter, r *http.Request) {
	params := GetContextParams(r)
	appSlug := mux.Vars(r)["appSlug"]

	if params.AppSlug != appSlug {
		logger.Error(errors.Errorf("app slug in path %s does not match app slug in context %s", appSlug, params.AppSlug))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	preflightData, err := upgradepreflight.GetPreflightData()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get preflight data"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var preflightResult preflighttypes.PreflightResult
	if preflightData.Result != nil {
		preflightResult = *preflightData.Result
	}

	response := GetPreflightResultResponse{
		PreflightResult:   preflightResult,
		PreflightProgress: preflightData.Progress,
	}
	JSON(w, 200, response)
}
