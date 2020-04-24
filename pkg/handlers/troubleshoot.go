package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kotsadm/pkg/app"
	"github.com/replicatedhq/kotsadm/pkg/license"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/replicatedhq/kotsadm/pkg/persistence"
	"github.com/replicatedhq/kotsadm/pkg/redact"
	"github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta1"
	"github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"github.com/replicatedhq/yaml/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

func init() {
	scheme.AddToScheme(scheme.Scheme)
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
		// this isn't fatal, since we can just omit the redact spec
		redactSpec = ""
		logger.Error(err)
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
	randomBundleID := strings.ToLower(rand.String(32))
	if r.Header.Get("Bundle-Upload-Host") != "" {
		uploadURL = fmt.Sprintf("%s/api/v1/troubleshoot/%s/%s", r.Header.Get("Bundle-Upload-Host"), foundApp.ID, randomBundleID)
	} else if inCluster == "true" {
		uploadURL = fmt.Sprintf("%s/api/v1/troubleshoot/%s/%s", fmt.Sprintf("http://kotsadm-api.%s.svc.cluster.local:3000", os.Getenv("POD_NAMESPACE")), foundApp.ID, randomBundleID)
	} else {
		uploadURL = fmt.Sprintf("%s/api/v1/troubleshoot/%s/%s", os.Getenv("API_ADVERTISE_ENDPOINT"), foundApp.ID, randomBundleID)
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
				URI:    uploadURL,
				Method: "PUT",
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
		// this isn't fatal, since we can just omit the redact spec
		redactSpec = ""
		logger.Error(err)
	}
	if redactSpec != "" {
		fullTroubleshoot = fmt.Sprintf("%s\n---\n%s", string(specBytes), redactSpec)
	}

	w.WriteHeader(200)
	w.Write([]byte(fullTroubleshoot))
}

// if a namespace is not set for a secret/run/logs/exec/copy collector, set it to the current namespace
func populateNamespaces(existingSpec *v1beta1.Collector) *v1beta1.Collector {
	if existingSpec == nil {
		return nil
	} else if existingSpec.Spec.Collectors == nil {
		return existingSpec
	}

	collects := []*v1beta1.Collect{}
	for _, collect := range existingSpec.Spec.Collectors {
		if collect.Secret != nil {
			if collect.Secret.Namespace == "" {
				collect.Secret.Namespace = os.Getenv("POD_NAMESPACE")
			}
		}
		if collect.Run != nil {
			if collect.Run.Namespace == "" {
				collect.Run.Namespace = os.Getenv("POD_NAMESPACE")
			}
		}
		if collect.Logs != nil {
			if collect.Logs.Namespace == "" {
				collect.Logs.Namespace = os.Getenv("POD_NAMESPACE")
			}
		}
		if collect.Exec != nil {
			if collect.Exec.Namespace == "" {
				collect.Exec.Namespace = os.Getenv("POD_NAMESPACE")
			}
		}
		if collect.Copy != nil {
			if collect.Copy.Namespace == "" {
				collect.Copy.Namespace = os.Getenv("POD_NAMESPACE")
			}
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
				Name:      "kots/admin_console",
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
