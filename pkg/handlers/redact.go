package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/redact"
	redacttypes "github.com/replicatedhq/kots/pkg/redact/types"
	"github.com/replicatedhq/kots/pkg/util"
)

type UpdateRedactRequest struct {
	RedactSpec    string `json:"redactSpec"`
	RedactSpecURL string `json:"redactSpecUrl"`
}

type UpdateRedactResponse struct {
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	UpdatedSpec string `json:"updatedSpec"`
}

type GetRedactResponse struct {
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	UpdatedSpec string `json:"updatedSpec"`
}

type GetRedactorResponse struct {
	Redactor string                   `json:"redactor"`
	Metadata redacttypes.RedactorList `json:"redactorMetadata"`

	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type ListRedactorsResponse struct {
	Redactors []redacttypes.RedactorList `json:"redactors"`

	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type PostRedactorMetadata struct {
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
	New         bool   `json:"new"`
	Redactor    string `json:"redactor"`
}

type PostRedactorEnabledMetadata struct {
	Enabled bool `json:"enabled"`
}

func (h *Handler) UpdateRedact(w http.ResponseWriter, r *http.Request) {
	updateRedactResponse := UpdateRedactResponse{
		Success: false,
	}

	updateRedactRequest := UpdateRedactRequest{}
	if err := json.NewDecoder(r.Body).Decode(&updateRedactRequest); err != nil {
		logger.Error(err)
		updateRedactResponse.Error = "failed to decode request body"
		JSON(w, 400, updateRedactResponse)
		return
	}

	setSpec := ""
	if updateRedactRequest.RedactSpec != "" {
		setSpec = updateRedactRequest.RedactSpec
	} else if updateRedactRequest.RedactSpecURL != "" {
		req, err := util.NewRequest("GET", updateRedactRequest.RedactSpecURL, nil)
		if err != nil {
			logger.Error(err)
			updateRedactResponse.Error = "failed to create request to get spec from url"
			JSON(w, 500, updateRedactResponse)
			return
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Error(err)
			updateRedactResponse.Error = "failed to get spec from url"
			JSON(w, 500, updateRedactResponse)
			return
		}
		defer resp.Body.Close()

		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error(err)
			updateRedactResponse.Error = "failed to read spec response from url"
			JSON(w, 500, updateRedactResponse)
			return
		}
		setSpec = string(respBytes)
	} else {
		updateRedactResponse.Error = "no spec or url provided"
		JSON(w, 400, updateRedactResponse)
		return
	}

	errMessage, err := redact.SetRedactSpec(setSpec)
	if err != nil {
		logger.Error(err)
		updateRedactResponse.Error = errMessage
		JSON(w, 500, updateRedactResponse)
		return
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		updateRedactResponse.Error = "failed to get kubernetes client"
		JSON(w, 500, updateRedactResponse)
		return
	}

	err = redact.GenerateKotsadmRedactSpec(clientset)
	if err != nil {
		logger.Error(err)
		updateRedactResponse.Error = "failed to generate kotsadm redact spec"
		JSON(w, 500, updateRedactResponse)
		return
	}

	data, errMessage, err := redact.GetRedactSpec()
	if err != nil {
		logger.Error(err)
		updateRedactResponse.Error = errMessage
		JSON(w, 500, updateRedactResponse)
	}

	updateRedactResponse.Success = true
	updateRedactResponse.UpdatedSpec = data
	JSON(w, 200, updateRedactResponse)
}

func (h *Handler) GetRedact(w http.ResponseWriter, r *http.Request) {
	getRedactResponse := GetRedactResponse{
		Success: false,
	}

	data, errMessage, err := redact.GetRedactSpec()
	if err != nil {
		logger.Error(err)
		getRedactResponse.Error = errMessage
		JSON(w, 500, getRedactResponse)
	}

	getRedactResponse.Success = true
	getRedactResponse.UpdatedSpec = data
	JSON(w, 200, getRedactResponse)
}

func (h *Handler) GetRedactMetadataAndYaml(w http.ResponseWriter, r *http.Request) {
	getRedactorResponse := GetRedactorResponse{
		Success: false,
	}

	redactorSlug := mux.Vars(r)["slug"]

	redactorObj, err := redact.GetRedactBySlug(redactorSlug)
	if err != nil {
		logger.Error(err)
		getRedactorResponse.Error = "failed to get redactor"
		JSON(w, http.StatusInternalServerError, getRedactorResponse)
		return
	}

	getRedactorResponse.Success = true
	getRedactorResponse.Redactor = redactorObj.Redact
	getRedactorResponse.Metadata = redactorObj.Metadata
	JSON(w, http.StatusOK, getRedactorResponse)
	return
}

func (h *Handler) ListRedactors(w http.ResponseWriter, r *http.Request) {
	listRedactorsResponse := ListRedactorsResponse{
		Success: false,
	}

	redactors, err := redact.GetRedactInfo()
	if err != nil {
		logger.Error(err)
		listRedactorsResponse.Error = "failed to get redactors"
		JSON(w, http.StatusInternalServerError, listRedactorsResponse)
		return
	}

	listRedactorsResponse.Success = true
	listRedactorsResponse.Redactors = redactors
	JSON(w, http.StatusOK, listRedactorsResponse)
	return
}

func (h *Handler) SetRedactMetadataAndYaml(w http.ResponseWriter, r *http.Request) {
	metadataResponse := GetRedactorResponse{
		Success: false,
	}

	redactorSlug := mux.Vars(r)["slug"]

	updateRedactRequest := PostRedactorMetadata{}
	if err := json.NewDecoder(r.Body).Decode(&updateRedactRequest); err != nil {
		logger.Error(err)
		metadataResponse.Error = "failed to decode request body"
		JSON(w, 400, metadataResponse)
		return
	}

	newRedactor, err := redact.SetRedactYaml(redactorSlug, updateRedactRequest.Description, updateRedactRequest.Enabled, updateRedactRequest.New, []byte(updateRedactRequest.Redactor))
	if err != nil {
		logger.Error(err)
		metadataResponse.Error = err.Error()
		JSON(w, 400, metadataResponse)
		return
	}

	metadataResponse.Success = true
	metadataResponse.Metadata = newRedactor.Metadata
	metadataResponse.Redactor = newRedactor.Redact
	JSON(w, http.StatusOK, metadataResponse)
	return
}

func (h *Handler) DeleteRedact(w http.ResponseWriter, r *http.Request) {
	redactorSlug := mux.Vars(r)["slug"]
	err := redact.DeleteRedact(redactorSlug)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (h *Handler) SetRedactEnabled(w http.ResponseWriter, r *http.Request) {
	metadataResponse := GetRedactorResponse{
		Success: false,
	}

	redactorSlug := mux.Vars(r)["slug"]

	updateRedactRequest := PostRedactorEnabledMetadata{}
	if err := json.NewDecoder(r.Body).Decode(&updateRedactRequest); err != nil {
		logger.Error(err)
		metadataResponse.Error = "failed to decode request body"
		JSON(w, 400, metadataResponse)
		return
	}

	updatedRedactor, err := redact.SetRedactEnabled(redactorSlug, updateRedactRequest.Enabled)
	if err != nil {
		logger.Error(err)
		metadataResponse.Error = "failed to update redactor status"
		JSON(w, 400, metadataResponse)
		return
	}

	metadataResponse.Success = true
	metadataResponse.Metadata = updatedRedactor.Metadata
	metadataResponse.Redactor = updatedRedactor.Redact
	JSON(w, http.StatusOK, metadataResponse)
	return
}
