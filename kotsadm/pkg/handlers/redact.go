package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/redact"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
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
	Redactor string              `json:"redactor"`
	Metadata redact.RedactorList `json:"redactorMetadata"`

	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type ListRedactorsResponse struct {
	Redactors []redact.RedactorList `json:"redactors"`

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

func UpdateRedact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	updateRedactResponse := UpdateRedactResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		updateRedactResponse.Error = "failed to parse authorization header"
		JSON(w, 401, updateRedactResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		updateRedactResponse.Error = "no session in auth header"
		JSON(w, 401, updateRedactResponse)
		return
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
		req, err := http.NewRequest("GET", updateRedactRequest.RedactSpecURL, nil)
		if err != nil {
			logger.Error(err)
			updateRedactResponse.Error = "failed to create request to get spec from url"
			JSON(w, 500, updateRedactResponse)
			return
		}

		req.Header.Set("User-Agent", "replicatedhq/kotsadm")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Error(err)
			updateRedactResponse.Error = "failed to get spec from url"
			JSON(w, 500, updateRedactResponse)
			return
		}
		defer resp.Body.Close()

		respBytes, err := ioutil.ReadAll(resp.Body)
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

func GetRedact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	getRedactResponse := GetRedactResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		getRedactResponse.Error = "failed to parse authorization header"
		JSON(w, 401, getRedactResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		getRedactResponse.Error = "no session in auth header"
		JSON(w, 401, getRedactResponse)
		return
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

func GetRedactMetadataAndYaml(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	getRedactorResponse := GetRedactorResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		getRedactorResponse.Error = "failed to parse authorization header"
		JSON(w, 401, getRedactorResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		getRedactorResponse.Error = "no session in auth header"
		JSON(w, 401, getRedactorResponse)
		return
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

func ListRedactors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	listRedactorsResponse := ListRedactorsResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		listRedactorsResponse.Error = "failed to parse authorization header"
		JSON(w, 401, listRedactorsResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		listRedactorsResponse.Error = "no session in auth header"
		JSON(w, 401, listRedactorsResponse)
		return
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

func SetRedactMetadataAndYaml(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	metadataResponse := GetRedactorResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		metadataResponse.Error = "failed to parse authorization header"
		JSON(w, 401, metadataResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		metadataResponse.Error = "no session in auth header"
		JSON(w, 401, metadataResponse)
		return
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

func DeleteRedact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(401)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		w.WriteHeader(401)
		return
	}

	redactorSlug := mux.Vars(r)["slug"]
	err = redact.DeleteRedact(redactorSlug)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func SetRedactEnabled(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	metadataResponse := GetRedactorResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		metadataResponse.Error = "failed to parse authorization header"
		JSON(w, 401, metadataResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		metadataResponse.Error = "no session in auth header"
		JSON(w, 401, metadataResponse)
		return
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
