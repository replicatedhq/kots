package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsutil"
	"github.com/replicatedhq/kots/kotsadm/pkg/license"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kots/kotsadm/pkg/redact"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshot"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	"github.com/replicatedhq/kots/pkg/template"
	troubleshootanalyze "github.com/replicatedhq/troubleshoot/pkg/analyze"
	"github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta1"
	troubleshootv1beta1 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta1"
	"github.com/replicatedhq/troubleshoot/pkg/convert"
	redact2 "github.com/replicatedhq/troubleshoot/pkg/redact"
	"github.com/replicatedhq/yaml/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/scheme"
)

type GetSupportBundleResponse struct {
	ID          string                       `json:"id"`
	Slug        string                       `json:"slug"`
	AppID       string                       `json:"appId"`
	Name        string                       `json:"name"`
	Size        float64                      `json:"size"`
	Status      string                       `json:"status"`
	TreeIndex   string                       `json:"treeIndex"`
	CreatedAt   time.Time                    `json:"createdAt"`
	UploadedAt  *time.Time                   `json:"uploadedAt"`
	IsArchived  bool                         `json:"isArchived"`
	LicenseType string                       `json:"licenseType,omitempty"`
	Analysis    *types.SupportBundleAnalysis `json:"analysis"`
}

type GetSupportBundleFilesResponse struct {
	Files map[string][]byte `json:"files"`

	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type ListSupportBundlesResponse struct {
	SupportBundles []ResponseSupportBundle `json:"supportBundles"`
}
type ResponseSupportBundle struct {
	ID          string                       `json:"id"`
	Slug        string                       `json:"slug"`
	AppID       string                       `json:"appId"`
	Name        string                       `json:"name"`
	Size        float64                      `json:"size"`
	Status      string                       `json:"status"`
	CreatedAt   time.Time                    `json:"createdAt"`
	UploadedAt  *time.Time                   `json:"uploadedAt"`
	IsArchived  bool                         `json:"isArchived"`
	LicenseType string                       `json:"licenseType,omitempty"`
	Analysis    *types.SupportBundleAnalysis `json:"analysis"`
}

type GetSupportBundleRedactionsResponse struct {
	Redactions redact2.RedactionList `json:"redactions"`

	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type PutSupportBundleRedactions struct {
	Redactions redact2.RedactionList `json:"redactions"`
}

func GetSupportBundle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

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

	bundleSlug := mux.Vars(r)["bundleSlug"]

	bundle, err := supportbundle.GetFromSlug(bundleSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	licenseType, err := supportbundle.GetLicenseType(bundle.ID)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get license type for bundle %s", bundle.Slug))
	}

	analysis, err := supportbundle.GetBundleAnalysis(bundle.ID)
	if err != nil {
		logger.Error(errors.Wrapf(err, "failed to get analysis for bundle %s", bundle.Slug))
	}

	getSupportBundleResponse := GetSupportBundleResponse{
		ID:          bundle.ID,
		Slug:        bundle.Slug,
		AppID:       bundle.AppID,
		Name:        bundle.Name,
		Size:        bundle.Size,
		Status:      bundle.Status,
		TreeIndex:   bundle.TreeIndex,
		CreatedAt:   bundle.CreatedAt,
		UploadedAt:  bundle.UploadedAt,
		IsArchived:  bundle.IsArchived,
		LicenseType: licenseType,
		Analysis:    analysis,
	}

	JSON(w, 200, getSupportBundleResponse)
}

func GetSupportBundleFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	getSupportBundleFilesResponse := GetSupportBundleFilesResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		getSupportBundleFilesResponse.Error = "failed to parse authorization header"
		JSON(w, 401, getSupportBundleFilesResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		getSupportBundleFilesResponse.Error = "no session in auth header"
		JSON(w, 401, getSupportBundleFilesResponse)
		return
	}

	bundleID := mux.Vars(r)["bundleId"]
	filenames := r.URL.Query()["filename"]

	files, err := supportbundle.GetFilesContents(bundleID, filenames)
	if err != nil {
		logger.Error(err)
		getSupportBundleFilesResponse.Error = "failed to get files"
		JSON(w, 500, getSupportBundleFilesResponse)
		return
	}

	getSupportBundleFilesResponse.Success = true
	getSupportBundleFilesResponse.Files = files

	JSON(w, 200, getSupportBundleFilesResponse)
}

func ListSupportBundles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

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

	appSlug := mux.Vars(r)["appSlug"]

	a, err := app.GetFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	supportBundles, err := supportbundle.List(a.ID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	responseSupportBundles := []ResponseSupportBundle{}
	for _, bundle := range supportBundles {
		licenseType, err := supportbundle.GetLicenseType(bundle.ID)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to get license type for bundle %s", bundle.Slug))
		}

		analysis, err := supportbundle.GetBundleAnalysis(bundle.ID)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to get analysis for bundle %s", bundle.Slug))
		}

		responseSupportBundle := ResponseSupportBundle{
			ID:          bundle.ID,
			Slug:        bundle.Slug,
			AppID:       bundle.AppID,
			Name:        bundle.Name,
			Size:        bundle.Size,
			Status:      bundle.Status,
			CreatedAt:   bundle.CreatedAt,
			UploadedAt:  bundle.UploadedAt,
			IsArchived:  bundle.IsArchived,
			LicenseType: licenseType,
			Analysis:    analysis,
		}

		responseSupportBundles = append(responseSupportBundles, responseSupportBundle)
	}

	listSupportBundlesResponse := ListSupportBundlesResponse{
		SupportBundles: responseSupportBundles,
	}

	JSON(w, 200, listSupportBundlesResponse)
}

func DownloadSupportBundle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		JSON(w, 401, nil)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		JSON(w, 401, nil)
		return
	}

	bundleID := mux.Vars(r)["bundleId"]

	bundle, err := supportbundle.GetSupportBundle(bundleID)
	if err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}
	defer os.RemoveAll(bundle)

	f, err := os.Open(bundle)
	if err != nil {
		logger.Error(err)
		JSON(w, 500, nil)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", `attachment; filename="supportbundle.tar.gz"`)
	io.Copy(w, f)
}

func UploadSupportBundle(w http.ResponseWriter, r *http.Request) {
	bundleContents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	tmpFile, err := ioutil.TempFile("", "kots")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	err = ioutil.WriteFile(tmpFile.Name(), bundleContents, 0644)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	supportBundle, err := supportbundle.CreateBundle(mux.Vars(r)["bundleId"], mux.Vars(r)["appId"], tmpFile.Name())
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	// we need the app archive to get the analyzers
	a, err := app.Get(mux.Vars(r)["appId"])
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	archiveDir, err := version.GetAppVersionArchive(a.ID, a.CurrentSequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(archiveDir)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	analyzer := kotsKinds.Analyzer
	if analyzer == nil {
		analyzer = &troubleshootv1beta1.Analyzer{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "troubleshoot.replicated.com/v1beta1",
				Kind:       "Analyzer",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "default-analyzers",
			},
			Spec: troubleshootv1beta1.AnalyzerSpec{
				Analyzers: []*troubleshootv1beta1.Analyze{},
			},
		}
	}

	if err := supportbundle.InjectDefaultAnalyzers(analyzer); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	s := k8sjson.NewYAMLSerializer(k8sjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(analyzer, &b); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	analyzeResult, err := troubleshootanalyze.DownloadAndAnalyze(tmpFile.Name(), b.String())
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	data := convert.FromAnalyzerResult(analyzeResult)
	insights, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	if err := supportbundle.SetBundleAnalysis(supportBundle.ID, insights); err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	fmt.Printf("analyzeResult %#v\n", analyzeResult)
}

func GetDefaultTroubleshoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	defaultTroubleshootSpec := addDefaultTroubleshoot(nil, "")
	defaultBytes, err := yaml.Marshal(defaultTroubleshootSpec)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	fullTroubleshoot := string(defaultBytes)
	redactSpec, _, err := redact.GetRedactSpec()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	if redactSpec != "" {
		fullTroubleshoot = fmt.Sprintf("%s\n---\n%s", string(defaultBytes), redactSpec)
	}

	w.WriteHeader(200)
	w.Write([]byte(fullTroubleshoot))
}

func GetTroubleshoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]
	inCluster := r.URL.Query().Get("incluster")

	// get app from slug
	foundApp, err := app.GetFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	// TODO get from watch ID, not just app id

	// get troubleshoot spec from db
	existingSpec, err := getAppTroubleshoot(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(existingSpec), nil, nil)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	existingTs := obj.(*v1beta1.Collector)

	existingTs = populateNamespaces(existingTs)

	// determine an upload URL
	var uploadURL string
	var redactURL string
	randomBundleID := strings.ToLower(rand.String(32))
	if r.Header.Get("Bundle-Upload-Host") != "" {
		uploadURL = fmt.Sprintf("%s/api/v1/troubleshoot/%s/%s", r.Header.Get("Bundle-Upload-Host"), foundApp.ID, randomBundleID)
		redactURL = fmt.Sprintf("%s/api/v1/troubleshoot/supportbundle/%s/redactions", r.Header.Get("Bundle-Upload-Host"), randomBundleID)
	} else if inCluster == "true" {
		uploadURL = fmt.Sprintf("%s/api/v1/troubleshoot/%s/%s", fmt.Sprintf("http://kotsadm-api.%s.svc.cluster.local:3000", os.Getenv("POD_NAMESPACE")), foundApp.ID, randomBundleID)
		redactURL = fmt.Sprintf("%s/api/v1/troubleshoot/supportbundle/%s/redactions", fmt.Sprintf("http://kotsadm-api.%s.svc.cluster.local:3000", os.Getenv("POD_NAMESPACE")), randomBundleID)
	} else {
		uploadURL = fmt.Sprintf("%s/api/v1/troubleshoot/%s/%s", os.Getenv("API_ADVERTISE_ENDPOINT"), foundApp.ID, randomBundleID)
		redactURL = fmt.Sprintf("%s/api/v1/troubleshoot/supportbundle/%s/redactions", os.Getenv("API_ADVERTISE_ENDPOINT"), randomBundleID)
	}

	licenseString, err := license.GetCurrentLicenseString(foundApp)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	tsSpec := addDefaultTroubleshoot(existingTs, licenseString)
	tsSpec.Spec.AfterCollection = []*v1beta1.AfterCollection{
		{
			UploadResultsTo: &v1beta1.ResultRequest{
				URI:       uploadURL,
				Method:    "PUT",
				RedactURI: redactURL,
			},
		},
	}

	specBytes, err := yaml.Marshal(tsSpec)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	fullTroubleshoot := string(specBytes)
	redactSpec, _, err := redact.GetRedactSpec()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	if redactSpec != "" {
		fullTroubleshoot = fmt.Sprintf("%s\n---\n%s", string(specBytes), redactSpec)
	}

	w.WriteHeader(200)
	w.Write([]byte(fullTroubleshoot))
}

func GetSupportBundleRedactions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	getSupportBundleRedactionsResponse := GetSupportBundleRedactionsResponse{
		Success: false,
	}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		getSupportBundleRedactionsResponse.Error = "failed to parse authorization header"
		JSON(w, 401, getSupportBundleRedactionsResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		getSupportBundleRedactionsResponse.Error = "no session in auth header"
		JSON(w, 401, getSupportBundleRedactionsResponse)
		return
	}

	bundleID := mux.Vars(r)["bundleId"]
	redactions, err := supportbundle.GetRedactions(bundleID)
	if err != nil {
		logger.Error(err)
		getSupportBundleRedactionsResponse.Error = fmt.Sprintf("failed to find redactions for bundle %s", bundleID)
		JSON(w, 400, getSupportBundleRedactionsResponse)
		return
	}

	getSupportBundleRedactionsResponse.Success = true
	getSupportBundleRedactionsResponse.Redactions = redactions

	JSON(w, 200, getSupportBundleRedactionsResponse)
}

func SetSupportBundleRedactions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	redactionsBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	redactions := PutSupportBundleRedactions{}
	err = json.Unmarshal(redactionsBody, &redactions)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(400)
		return
	}

	bundleID := mux.Vars(r)["bundleId"]
	err = supportbundle.SetRedactions(bundleID, redactions.Redactions)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(201)
	return
}

// if a namespace is not set for a secret/run/logs/exec/copy collector, set it to the current namespace
func populateNamespaces(existingSpec *v1beta1.Collector) *v1beta1.Collector {
	if existingSpec == nil {
		return nil
	} else if existingSpec.Spec.Collectors == nil {
		return existingSpec
	}

	builder := template.Builder{}
	builder.AddCtx(template.StaticCtx{})

	ns := func(ns string) string {
		templated, err := builder.RenderTemplate("ns", ns)
		if err != nil {
			logger.Error(err)
		}
		if templated != "" {
			return templated
		}
		return os.Getenv("POD_NAMESPACE")
	}

	collects := []*v1beta1.Collect{}
	for _, collect := range existingSpec.Spec.Collectors {
		if collect.Secret != nil {
			collect.Secret.Namespace = ns(collect.Secret.Namespace)
		}
		if collect.Run != nil {
			collect.Run.Namespace = ns(collect.Run.Namespace)
		}
		if collect.Logs != nil {
			collect.Logs.Namespace = ns(collect.Logs.Namespace)
		}
		if collect.Exec != nil {
			collect.Exec.Namespace = ns(collect.Exec.Namespace)
		}
		if collect.Copy != nil {
			collect.Copy.Namespace = ns(collect.Copy.Namespace)
		}
		collects = append(collects, collect)
	}
	existingSpec.Spec.Collectors = collects
	return existingSpec
}

func addDefaultTroubleshoot(existingSpec *v1beta1.Collector, licenseData string) *v1beta1.Collector {
	if existingSpec == nil {
		existingSpec = &v1beta1.Collector{
			TypeMeta: v1.TypeMeta{
				Kind:       "Collector",
				APIVersion: "troubleshoot.replicated.com/v1beta1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "default-collector",
			},
		}
	}

	existingSpec.Spec.Collectors = append(existingSpec.Spec.Collectors, []*v1beta1.Collect{
		{
			Data: &v1beta1.Data{
				CollectorMeta: v1beta1.CollectorMeta{
					CollectorName: "license.yaml",
				},
				Name: "kots/admin-console",
				Data: licenseData,
			},
		},
		{
			Secret: &v1beta1.Secret{
				CollectorMeta: v1beta1.CollectorMeta{
					CollectorName: "kotsadm-replicated-registry",
				},
				SecretName:   "kotsadm-replicated-registry",
				Namespace:    os.Getenv("POD_NAMESPACE"),
				Key:          ".dockerconfigjson",
				IncludeValue: false,
			},
		},
	}...)
	existingSpec.Spec.Collectors = append(existingSpec.Spec.Collectors, makeDbCollectors()...)
	existingSpec.Spec.Collectors = append(existingSpec.Spec.Collectors, makeKotsadmCollectors()...)
	existingSpec.Spec.Collectors = append(existingSpec.Spec.Collectors, makeRookCollectors()...)
	existingSpec.Spec.Collectors = append(existingSpec.Spec.Collectors, makeKurlCollectors()...)
	existingSpec.Spec.Collectors = append(existingSpec.Spec.Collectors, makeVeleroCollectors()...)
	return existingSpec
}

func makeDbCollectors() []*v1beta1.Collect {
	dbCollectors := []*v1beta1.Collect{}

	pgConnectionString := os.Getenv("POSTGRES_URI")
	parsedPg, err := url.Parse(pgConnectionString)
	if err == nil {
		username := "kotsadm"
		if parsedPg.User != nil {
			username = parsedPg.User.Username()
		}
		dbCollectors = append(dbCollectors, &v1beta1.Collect{
			Exec: &v1beta1.Exec{
				CollectorMeta: v1beta1.CollectorMeta{
					CollectorName: "kotsadm-postgres-db",
				},
				Name:          "kots/admin-console",
				Selector:      []string{fmt.Sprintf("app=%s", parsedPg.Host)},
				Namespace:     os.Getenv("POD_NAMESPACE"),
				ContainerName: parsedPg.Host,
				Command:       []string{"pg_dump"},
				Args:          []string{"-U", username},
				Timeout:       "10s",
			},
		})
	}
	return dbCollectors
}

func makeKotsadmCollectors() []*v1beta1.Collect {
	names := []string{
		"kotsadm-postgres",
		"kotsadm",
		"kotsadm-api",
		"kotsadm-operator",
		"kurl-proxy-kotsadm",
	}
	rookCollectors := []*v1beta1.Collect{}
	for _, name := range names {
		rookCollectors = append(rookCollectors, &v1beta1.Collect{
			Logs: &v1beta1.Logs{
				CollectorMeta: v1beta1.CollectorMeta{
					CollectorName: name,
				},
				Name:      "kots/admin-console",
				Selector:  []string{fmt.Sprintf("app=%s", name)},
				Namespace: os.Getenv("POD_NAMESPACE"),
			},
		})
	}
	return rookCollectors
}

func makeRookCollectors() []*v1beta1.Collect {
	names := []string{
		"rook-ceph-agent",
		"rook-ceph-mgr",
		"rook-ceph-mon",
		"rook-ceph-operator",
		"rook-ceph-osd",
		"rook-ceph-osd-prepare",
		"rook-ceph-rgw",
		"rook-discover",
	}
	rookCollectors := []*v1beta1.Collect{}
	for _, name := range names {
		rookCollectors = append(rookCollectors, &v1beta1.Collect{
			Logs: &v1beta1.Logs{
				CollectorMeta: v1beta1.CollectorMeta{
					CollectorName: name,
				},
				Name:      "kots/rook",
				Selector:  []string{fmt.Sprintf("app=%s", name)},
				Namespace: "rook-ceph",
			},
		})
	}
	return rookCollectors
}

func makeKurlCollectors() []*v1beta1.Collect {
	names := []string{
		"registry",
	}
	rookCollectors := []*v1beta1.Collect{}
	for _, name := range names {
		rookCollectors = append(rookCollectors, &v1beta1.Collect{
			Logs: &v1beta1.Logs{
				CollectorMeta: v1beta1.CollectorMeta{
					CollectorName: name,
				},
				Name:      "kots/kurl",
				Selector:  []string{fmt.Sprintf("app=%s", name)},
				Namespace: "kurl",
			},
		})
	}
	return rookCollectors
}

func makeVeleroCollectors() []*v1beta1.Collect {
	collectors := []*v1beta1.Collect{}

	veleroNamespace, err := snapshot.DetectVeleroNamespace()
	if err != nil {
		logger.Error(err)
		return collectors
	}

	if veleroNamespace == "" {
		return collectors
	}

	selectors := []string{
		"component=velero",
		"app.kubernetes.io/name=velero",
	}

	for _, selector := range selectors {
		collectors = append(collectors, &v1beta1.Collect{
			Logs: &v1beta1.Logs{
				CollectorMeta: v1beta1.CollectorMeta{
					CollectorName: "velero",
				},
				Name:      "velero",
				Selector:  []string{selector},
				Namespace: veleroNamespace,
			},
		})
	}

	return collectors
}

func getAppTroubleshoot(slug string) (string, error) {
	q := `select supportbundle_spec from app_version
      inner join app on app_version.app_id = app.id and app_version.sequence = app.current_sequence
      where app.slug = $1`

	spec := ""

	db := persistence.MustGetPGSession()
	row := db.QueryRow(q, slug)
	err := row.Scan(&spec)
	if err != nil {
		return "", err
	}
	return spec, nil
}
