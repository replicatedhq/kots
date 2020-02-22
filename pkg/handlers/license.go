package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	kotspull "github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/session"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme")

type SyncLicenseRequest struct {
	LicenseData string `json:"licenseData"`
}

type SyncLicenseResponse struct {
	ID              string                `json:"id"`
	ExpiresAt       time.Time             `json:"expiresAt"`
	ChannelName     string                `json:"channelName"`
	LicenseSequence int64                 `json:"licenseSequence"`
	LicenseType     string                `json:"licenseType"`
	Entitlements    []EntitlementResponse `json:"entitlements"`
}

type EntitlementResponse struct {
	Title string      `json:"title"`
	Value interface{} `json:"value"`
	Label string      `json:"label"`
}

func SyncLicense(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	syncLicenseRequest := SyncLicenseRequest{}
	if err := json.NewDecoder(r.Body).Decode(&syncLicenseRequest); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		w.WriteHeader(401)
		return
	}

	foundApp, err := app.GetFromSlug(mux.Vars(r)["appSlug"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	archiveDir, err := app.GetAppVersionArchive(foundApp.ID, foundApp.CurrentSequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	defer os.RemoveAll(archiveDir)

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if kotsKinds.License == nil {
		logger.Error(errors.New("app does not have a license"))
		w.WriteHeader(500)
		return
	}

	latestLicense := kotsKinds.License
	if syncLicenseRequest.LicenseData != "" {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(syncLicenseRequest.LicenseData), nil, nil)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}

		unverifiedLicense := obj.(*kotsv1beta1.License)
		verifiedLicense, err := kotspull.VerifySignature(unverifiedLicense)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}

		latestLicense = verifiedLicense
	} else {
		// get from the api
		updatedLicense, err := kotslicense.GetLatestLicense(kotsKinds.License)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}
		latestLicense = updatedLicense
	}

	// Save and make a new version if the sequence has changed
	if latestLicense.Spec.LicenseSequence != kotsKinds.License.Spec.LicenseSequence {
		s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
		var b bytes.Buffer
		if err := s.Encode(latestLicense, &b); err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}
		if err := ioutil.WriteFile(filepath.Join(archiveDir, "upstream", "userdata", "license.yaml"), b.Bytes(), 0644); err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}

		newSequence, err := foundApp.CreateVersion(archiveDir, "License Change")
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}

		if err := app.CreateAppVersionArchive(foundApp.ID, newSequence, archiveDir); err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}
	}

	syncLicenseResponse := SyncLicenseResponse{
		ID:              latestLicense.Spec.LicenseID,
		ChannelName:     latestLicense.Spec.ChannelName,
		LicenseSequence: latestLicense.Spec.LicenseSequence,
		LicenseType:     latestLicense.Spec.LicenseType,
		Entitlements:    []EntitlementResponse{},
	}

	for key, entititlement := range latestLicense.Spec.Entitlements {
		if key == "expires_at" {
			if entititlement.Value.StrVal == "" {
				continue
			}

			expiration, err := time.Parse(time.RFC3339, entititlement.Value.StrVal)
			if err != nil {
				logger.Error(err)
				w.WriteHeader(500)
				return
			}
			syncLicenseResponse.ExpiresAt = expiration
		} else if key == "gitops_enabled" {
			/* do nothing */
		} else {
			syncLicenseResponse.Entitlements = append(syncLicenseResponse.Entitlements,
				EntitlementResponse{
					Title: entititlement.Title,
					Label: key,
					Value: entititlement.Value.Value(),
				})
		}
	}

	JSON(w, 200, syncLicenseResponse)
}
