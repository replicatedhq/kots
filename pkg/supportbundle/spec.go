package supportbundle

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotstypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	license "github.com/replicatedhq/kots/pkg/kotsadmlicense"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/registry"
	"github.com/replicatedhq/kots/pkg/render/helper"
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/template"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"go.uber.org/multierr"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

type DefaultTroubleshootOpts struct {
	Origin        string
	InCluster     bool
	DisableUpload bool
}

// CreateRenderedSpec creates the support bundle specification from defaults and the kots app
func CreateRenderedSpec(appID string, sequence int64, kotsKinds *kotsutil.KotsKinds, opts DefaultTroubleshootOpts) (*troubleshootv1beta2.SupportBundle, error) {
	builtBundle := kotsKinds.SupportBundle.DeepCopy()
	if builtBundle == nil {
		builtBundle = &troubleshootv1beta2.SupportBundle{
			TypeMeta: v1.TypeMeta{
				Kind:       "SupportBundle",
				APIVersion: "troubleshoot.sh/v1beta2",
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "default-supportbundle",
			},
		}

		if kotsKinds.Collector != nil {
			builtBundle.Spec.Collectors = kotsKinds.Collector.DeepCopy().Spec.Collectors
		}
		if kotsKinds.Analyzer != nil {
			builtBundle.Spec.Analyzers = kotsKinds.Analyzer.DeepCopy().Spec.Analyzers
		}
	}

	app, err := store.GetStore().GetApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}

	err = injectDefaults(app, builtBundle, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to inject defaults")
	}

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(builtBundle, &b); err != nil {
		return nil, errors.Wrap(err, "failed to encode support bundle")
	}

	templatedSpec := b.Bytes()

	renderedSpec, err := helper.RenderAppFile(app, &sequence, templatedSpec, kotsKinds, os.Getenv("POD_NAMESPACE"))
	if err != nil {
		return nil, errors.Wrap(err, "failed render support bundle spec")
	}

	// unmarshal the spec, look for image replacements in collectors and then remarshal
	// we do this after template rendering to support templating and then replacement
	supportBundle, err := kotsutil.LoadSupportBundleFromContents(renderedSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal rendered support bundle spec")
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get registry settings for app")
	}

	collectors, err := registry.UpdateCollectorSpecsWithRegistryData(supportBundle.Spec.Collectors, registrySettings, kotsKinds.Installation.Spec.KnownImages, kotsKinds.License)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update collectors")
	}
	supportBundle.Spec.Collectors = collectors
	b.Reset()
	if err := s.Encode(supportBundle, &b); err != nil {
		return nil, errors.Wrap(err, "failed to encode support bundle")
	}

	renderedSpec = b.Bytes()

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	secretName := GetSpecSecretName(app.Slug)

	existingSecret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to read support bundle secret")
	} else if kuberneteserrors.IsNotFound(err) {
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: os.Getenv("POD_NAMESPACE"),
				Labels:    kotstypes.GetKotsadmLabels(),
			},
			Data: map[string][]byte{
				SpecDataKey: renderedSpec,
			},
		}

		_, err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create support bundle secret")
		}

		return supportBundle, nil
	}

	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}
	existingSecret.Data[SpecDataKey] = renderedSpec
	existingSecret.ObjectMeta.Labels = kotstypes.GetKotsadmLabels()

	_, err = clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to update support bundle secret")
	}

	return supportBundle, nil
}

// injectDefaults injects the kots adm default collectors/analyzers in the the support bundle specification.
func injectDefaults(app *apptypes.App, supportBundle *troubleshootv1beta2.SupportBundle, opts DefaultTroubleshootOpts) error {
	populateNamespaces(supportBundle)

	addDefaultTroubleshoot(supportBundle, app)

	// determine an upload URL
	var uploadURL string
	var redactURL string
	randomBundleID := strings.ToLower(rand.String(32))
	if opts.DisableUpload {
		//Just use the library internally
		return nil
	} else if opts.Origin != "" {
		uploadURL = fmt.Sprintf("%s/api/v1/troubleshoot/%s/%s", opts.Origin, app.ID, randomBundleID)
		redactURL = fmt.Sprintf("%s/api/v1/troubleshoot/supportbundle/%s/redactions", opts.Origin, randomBundleID)
	} else if opts.InCluster {
		uploadURL = fmt.Sprintf("%s/api/v1/troubleshoot/%s/%s", fmt.Sprintf("http://kotsadm.%s.svc.cluster.local:3000", os.Getenv("POD_NAMESPACE")), app.ID, randomBundleID)
		redactURL = fmt.Sprintf("%s/api/v1/troubleshoot/supportbundle/%s/redactions", fmt.Sprintf("http://kotsadm.%s.svc.cluster.local:3000", os.Getenv("POD_NAMESPACE")), randomBundleID)
	} else {
		uploadURL = fmt.Sprintf("%s/api/v1/troubleshoot/%s/%s", os.Getenv("API_ADVERTISE_ENDPOINT"), app.ID, randomBundleID)
		redactURL = fmt.Sprintf("%s/api/v1/troubleshoot/supportbundle/%s/redactions", os.Getenv("API_ADVERTISE_ENDPOINT"), randomBundleID)
	}

	supportBundle.Spec.AfterCollection = []*troubleshootv1beta2.AfterCollection{
		{
			UploadResultsTo: &troubleshootv1beta2.ResultRequest{
				URI:       uploadURL,
				Method:    "PUT",
				RedactURI: redactURL,
			},
		},
	}

	return nil
}

// if a namespace is not set for a secret/run/logs/exec/copy collector, set it to the current namespace
func populateNamespaces(supportBundle *troubleshootv1beta2.SupportBundle) {
	if supportBundle == nil || supportBundle.Spec.Collectors == nil {
		return
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

	collects := []*troubleshootv1beta2.Collect{}
	for _, collect := range supportBundle.Spec.Collectors {
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
	supportBundle.Spec.Collectors = collects
}

func addDefaultTroubleshoot(supportBundle *troubleshootv1beta2.SupportBundle, app *apptypes.App) *troubleshootv1beta2.SupportBundle {
	if supportBundle.Spec.Collectors == nil {
		supportBundle.Spec.Collectors = make([]*troubleshootv1beta2.Collect, 0)
	}
	if supportBundle.Spec.Analyzers == nil {
		supportBundle.Spec.Analyzers = make([]*troubleshootv1beta2.Analyze, 0)
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Errorf("Failed to get kubernetes clientset: %v", err)
	}

	var image string
	var pullSecret *troubleshootv1beta2.ImagePullSecrets
	if clientset != nil {
		image, pullSecret, err = getImageAndSecret(context.TODO(), clientset)
		if err != nil {
			logger.Errorf("Failed to get kotsadm image and secret: %v", err)
		}
	}

	licenseData, err := license.GetCurrentLicenseString(app)
	if err != nil {
		logger.Errorf("Failed to load license data: %v", err)
	}

	if licenseData != "" {
		supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, &troubleshootv1beta2.Collect{
			Data: &troubleshootv1beta2.Data{
				CollectorMeta: troubleshootv1beta2.CollectorMeta{
					CollectorName: "license.yaml",
				},
				Name: "kots/admin-console",
				Data: licenseData,
			},
		})
	}

	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, &troubleshootv1beta2.Collect{
		Data: &troubleshootv1beta2.Data{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: "namespace.txt",
			},
			Name: "kots/admin-console",
			Data: os.Getenv("POD_NAMESPACE"),
		},
	})

	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, &troubleshootv1beta2.Collect{
		Secret: &troubleshootv1beta2.Secret{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: "kotsadm-replicated-registry",
			},
			Name:         "kotsadm-replicated-registry",
			Namespace:    os.Getenv("POD_NAMESPACE"),
			Key:          ".dockerconfigjson",
			IncludeValue: false,
		},
	})

	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, makeDbCollectors()...)
	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, makeKotsadmCollectors()...)
	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, makeGoRoutineCollectors()...)
	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, makeRookCollectors()...)
	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, makeKurlCollectors(image, pullSecret)...)
	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, makeVeleroCollectors()...)

	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, makeWeaveCollectors()...)
	supportBundle.Spec.Analyzers = append(supportBundle.Spec.Analyzers, makeWeaveAnalyzers()...)

	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, makeGoldpingerCollectors()...)
	supportBundle.Spec.Analyzers = append(supportBundle.Spec.Analyzers, makeGoldpingerAnalyzers()...)

	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, makeLonghornCollectors()...)
	supportBundle.Spec.Analyzers = append(supportBundle.Spec.Analyzers, makeLonghornAnalyzers()...)

	apps := []*apptypes.App{}
	if app != nil {
		apps = append(apps, app)
	} else {
		var err error
		apps, err = store.GetStore().ListInstalledApps()
		if err != nil {
			logger.Errorf("Failed to list installed apps: %v", err)
		}
	}

	if len(apps) > 0 {
		collectors, err := makeAppVersionArchiveCollectors(apps)
		if err != nil {
			logger.Errorf("Failed to make app version archive collectors: %v", err)
		}
		supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, collectors...)
	}

	supportBundle.Spec.Collectors = append(supportBundle.Spec.Collectors, makeCollectDCollectors(image, pullSecret)...)

	return supportBundle
}

func makeDbCollectors() []*troubleshootv1beta2.Collect {
	dbCollectors := []*troubleshootv1beta2.Collect{}

	pgConnectionString := os.Getenv("POSTGRES_URI")
	parsedPg, err := url.Parse(pgConnectionString)
	if err == nil {
		username := "kotsadm"
		if parsedPg.User != nil {
			username = parsedPg.User.Username()
		}
		dbCollectors = append(dbCollectors, &troubleshootv1beta2.Collect{
			Exec: &troubleshootv1beta2.Exec{
				CollectorMeta: troubleshootv1beta2.CollectorMeta{
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

func makeKotsadmCollectors() []*troubleshootv1beta2.Collect {
	names := []string{
		"kotsadm-postgres",
		"kotsadm",
		"kotsadm-operator",
		"kurl-proxy-kotsadm",
		"kotsadm-dex",
		"kotsadm-fs-minio",
		"kotsadm-s3-ops",
	}
	kotsadmCollectors := []*troubleshootv1beta2.Collect{}
	for _, name := range names {
		kotsadmCollectors = append(kotsadmCollectors, &troubleshootv1beta2.Collect{
			Logs: &troubleshootv1beta2.Logs{
				CollectorMeta: troubleshootv1beta2.CollectorMeta{
					CollectorName: name,
				},
				Name:      "kots/admin-console",
				Selector:  []string{fmt.Sprintf("app=%s", name)},
				Namespace: os.Getenv("POD_NAMESPACE"),
			},
		})
	}
	return kotsadmCollectors
}

func makeGoRoutineCollectors() []*troubleshootv1beta2.Collect {
	names := []string{
		"kotsadm",
		"kotsadm-operator",
	}
	goroutineCollectors := []*troubleshootv1beta2.Collect{}
	for _, name := range names {
		goroutineCollectors = append(goroutineCollectors, &troubleshootv1beta2.Collect{
			Exec: &troubleshootv1beta2.Exec{
				CollectorMeta: troubleshootv1beta2.CollectorMeta{
					CollectorName: fmt.Sprintf("%s-goroutines", name),
				},
				Name:          "kots/admin-console",
				Selector:      []string{fmt.Sprintf("app=%s", name)},
				Namespace:     os.Getenv("POD_NAMESPACE"),
				ContainerName: name,
				Command:       []string{"curl"},
				Args:          []string{"http://localhost:3030/goroutines"},
				Timeout:       "10s",
			},
		})
	}
	return goroutineCollectors
}

func makeRookCollectors() []*troubleshootv1beta2.Collect {
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
	rookCollectors := []*troubleshootv1beta2.Collect{}
	for _, name := range names {
		rookCollectors = append(rookCollectors, &troubleshootv1beta2.Collect{
			Logs: &troubleshootv1beta2.Logs{
				CollectorMeta: troubleshootv1beta2.CollectorMeta{
					CollectorName: name,
				},
				Name:      "kots/rook",
				Selector:  []string{fmt.Sprintf("app=%s", name)},
				Namespace: "rook-ceph",
			},
		})
	}

	rookCollectors = append(rookCollectors, &troubleshootv1beta2.Collect{
		Ceph: &troubleshootv1beta2.Ceph{},
	})

	return rookCollectors
}

func makeKurlCollectors(image string, pullSecret *troubleshootv1beta2.ImagePullSecrets) []*troubleshootv1beta2.Collect {
	collectors := []*troubleshootv1beta2.Collect{}

	if !kurl.IsKurl() {
		return collectors
	}

	names := []string{
		"registry",
		"ekc-operator",
	}
	for _, name := range names {
		collectors = append(collectors, &troubleshootv1beta2.Collect{
			Logs: &troubleshootv1beta2.Logs{
				CollectorMeta: troubleshootv1beta2.CollectorMeta{
					CollectorName: name,
				},
				Name:      "kots/kurl",
				Selector:  []string{fmt.Sprintf("app=%s", name)},
				Namespace: "kurl",
			},
		})
	}

	if image != "" {
		collectors = append(collectors, &troubleshootv1beta2.Collect{
			CopyFromHost: &troubleshootv1beta2.CopyFromHost{
				CollectorMeta: troubleshootv1beta2.CollectorMeta{
					CollectorName: "kurl-host-preflights",
				},
				Name:            "kots/kurl/host-preflights",
				HostPath:        "/var/lib/kurl/host-preflights",
				ExtractArchive:  true,
				Namespace:       os.Getenv("POD_NAMESPACE"),
				Image:           image,
				ImagePullSecret: pullSecret,
				ImagePullPolicy: string(corev1.PullIfNotPresent),
				Timeout:         "1m",
			},
		})
	}

	collectors = append(collectors, &troubleshootv1beta2.Collect{
		ConfigMap: &troubleshootv1beta2.ConfigMap{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: "kurl-current-config",
			},
			Name:           "kurl-current-config",
			Namespace:      "kurl",
			IncludeAllData: true,
		},
	})

	collectors = append(collectors, &troubleshootv1beta2.Collect{
		ConfigMap: &troubleshootv1beta2.ConfigMap{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: "kurl-last-config",
			},
			Name:           "kurl-last-config",
			Namespace:      "kurl",
			IncludeAllData: true,
		},
	})

	return collectors
}

func makeWeaveCollectors() []*troubleshootv1beta2.Collect {
	collectors := []*troubleshootv1beta2.Collect{}

	collectors = append(collectors, &troubleshootv1beta2.Collect{
		Exec: &troubleshootv1beta2.Exec{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: "weave-status",
			},
			Name:          "kots/kurl/weave",
			Selector:      []string{"name=weave-net"},
			Namespace:     "kube-system",
			ContainerName: "weave",
			Command:       []string{"/home/weave/weave"},
			Args:          []string{"--local", "status"},
			Timeout:       "10s",
		},
	})

	collectors = append(collectors, &troubleshootv1beta2.Collect{
		Exec: &troubleshootv1beta2.Exec{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: "weave-report",
			},
			Name:          "kots/kurl/weave",
			Selector:      []string{"name=weave-net"},
			Namespace:     "kube-system",
			ContainerName: "weave",
			Command:       []string{"/home/weave/weave"},
			Args:          []string{"--local", "report"},
			Timeout:       "10s",
		},
	})

	return collectors
}

func makeWeaveAnalyzers() []*troubleshootv1beta2.Analyze {
	analyzers := []*troubleshootv1beta2.Analyze{}

	analyzers = append(analyzers, &troubleshootv1beta2.Analyze{
		TextAnalyze: &troubleshootv1beta2.TextAnalyze{
			AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
				CheckName: "Weave Status",
			},
			FileName:     "kots/kurl/weave/kube-system/weave-net-*/weave-status-stdout.txt",
			RegexPattern: `Status: ready`,
			Outcomes: []*troubleshootv1beta2.Outcome{
				{
					Fail: &troubleshootv1beta2.SingleOutcome{
						Message: "Weave is not ready",
					},
				},
				{
					Pass: &troubleshootv1beta2.SingleOutcome{
						Message: "Weave is ready",
					},
				},
			},
		},
	})

	analyzers = append(analyzers, &troubleshootv1beta2.Analyze{
		TextAnalyze: &troubleshootv1beta2.TextAnalyze{

			AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
				CheckName: "Weave Report",
			},
			FileName:     "kots/kurl/weave/kube-system/weave-net-*/weave-report-stdout.txt",
			RegexPattern: `"Ready": true`,
			Outcomes: []*troubleshootv1beta2.Outcome{
				{
					Fail: &troubleshootv1beta2.SingleOutcome{
						Message: "Weave is not ready",
					},
				},
				{
					Pass: &troubleshootv1beta2.SingleOutcome{
						Message: "Weave is ready",
					},
				},
			},
		},
	})

	return analyzers
}

func makeVeleroCollectors() []*troubleshootv1beta2.Collect {
	collectors := []*troubleshootv1beta2.Collect{}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		return collectors
	}

	veleroNamespace, err := snapshot.DetectVeleroNamespace(context.TODO(), clientset, os.Getenv("POD_NAMESPACE"))
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
		collectors = append(collectors, &troubleshootv1beta2.Collect{
			Logs: &troubleshootv1beta2.Logs{
				CollectorMeta: troubleshootv1beta2.CollectorMeta{
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

func makeAppVersionArchiveCollectors(apps []*apptypes.App) ([]*troubleshootv1beta2.Collect, error) {
	dirPrefix, err := ioutil.TempDir("", "app-version-archive")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	go func() {
		<-time.After(10 * time.Minute)
		_ = os.RemoveAll(dirPrefix)
	}()

	collectors := []*troubleshootv1beta2.Collect{}
	for _, app := range apps {
		collector, err := makeAppVersionArchiveCollector(app, dirPrefix)
		if err != nil {
			err = multierr.Append(err, errors.Wrapf(err, "failed to make collector for app %s", app.Slug))
		} else {
			collectors = append(collectors, collector)
		}
	}

	return collectors, err
}

func makeAppVersionArchiveCollector(app *apptypes.App, dirPrefix string) (*troubleshootv1beta2.Collect, error) {
	fileName := filepath.Join(dirPrefix, fmt.Sprintf("%s.tar", app.Slug))
	archive, err := os.Create(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create temp file %s", fileName)
	}

	tempPath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tempPath)

	err = store.GetStore().GetAppVersionArchive(app.ID, app.CurrentSequence, tempPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app version archive")
	}

	tarWriter := tar.NewWriter(archive)
	defer tarWriter.Close()

	// only upstream includes files that don't contain any secrets
	archivePath := filepath.Join(tempPath, "upstream")

	err = filepath.Walk(archivePath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if archivePath == path {
			// root dir itself is the first item in the walk
			return nil
		}

		trimmedPath := strings.TrimPrefix(path, archivePath)

		// do not include userdata in archive
		if filepath.HasPrefix(trimmedPath, "/userdata") {
			return nil
		}

		tarHeader, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return errors.Wrapf(err, "failed to get tar header from file info header for file %s", trimmedPath)
		}
		tarHeader.Name = trimmedPath

		if err := tarWriter.WriteHeader(tarHeader); err != nil {
			return errors.Wrapf(err, "failed to write tar header for file %s", trimmedPath)
		}

		if fi.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return errors.Wrapf(err, "failed to open file %s", trimmedPath)
		}
		defer file.Close()

		if _, err := io.Copy(tarWriter, file); err != nil {
			return errors.Wrapf(err, "failed to write file %s contents", trimmedPath)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to walk archive dir %s", archivePath)
	}

	return &troubleshootv1beta2.Collect{
		Copy: &troubleshootv1beta2.Copy{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: fmt.Sprintf("spec-%s", app.Slug),
			},
			Selector: []string{
				"app=kotsadm", // can we assume this?
			},
			Namespace:     os.Getenv("POD_NAMESPACE"),
			ContainerName: "kotsadm", // can we assume this? kotsadm-api
			ContainerPath: fileName,
			Name:          fmt.Sprintf("kots/admin-console/app/%s", app.Slug),
		},
	}, nil
}

func makeCollectDCollectors(imageName string, pullSecret *troubleshootv1beta2.ImagePullSecrets) []*troubleshootv1beta2.Collect {
	collectors := []*troubleshootv1beta2.Collect{}

	if !kurl.IsKurl() {
		return collectors
	}

	if imageName != "" {
		collectors = append(collectors, &troubleshootv1beta2.Collect{
			Collectd: &troubleshootv1beta2.Collectd{
				CollectorMeta: troubleshootv1beta2.CollectorMeta{
					CollectorName: "collectd",
				},
				Namespace:       os.Getenv("POD_NAMESPACE"),
				Image:           imageName,
				ImagePullSecret: pullSecret,
				ImagePullPolicy: string(corev1.PullIfNotPresent),
				HostPath:        "/var/lib/collectd/rrd",
				Timeout:         "5m",
			},
		})
	}

	return collectors
}

func getImageAndSecret(ctx context.Context, clientset kubernetes.Interface) (imageName string, pullSecret *troubleshootv1beta2.ImagePullSecrets, err error) {
	namespace := os.Getenv("POD_NAMESPACE")

	var containers []corev1.Container
	var imagePullSecrets []corev1.LocalObjectReference
	if os.Getenv("POD_OWNER_KIND") == "deployment" {
		existingDeployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
		if err != nil {
			return imageName, pullSecret, errors.Wrap(err, "failed to get existing deployment")
		}
		imagePullSecrets = existingDeployment.Spec.Template.Spec.ImagePullSecrets
		containers = existingDeployment.Spec.Template.Spec.Containers
	} else {
		existingStatefulSet, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
		if err != nil {
			return imageName, pullSecret, errors.Wrap(err, "failed to get existing statefulset")
		}
		imagePullSecrets = existingStatefulSet.Spec.Template.Spec.ImagePullSecrets
		containers = existingStatefulSet.Spec.Template.Spec.Containers
	}

	for _, container := range containers {
		if container.Name == "kotsadm" {
			imageName = container.Image
			break
		}
	}
	if imageName == "" {
		return imageName, pullSecret, errors.New("container not found")
	}

	if len(imagePullSecrets) > 0 {
		existingSecret := imagePullSecrets[0]
		pullSecret = &troubleshootv1beta2.ImagePullSecrets{
			Name: existingSecret.Name,
		}
	}

	return imageName, pullSecret, nil
}

func makeGoldpingerCollectors() []*troubleshootv1beta2.Collect {
	collectors := []*troubleshootv1beta2.Collect{}

	collectors = append(collectors, &troubleshootv1beta2.Collect{
		Exec: &troubleshootv1beta2.Exec{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: "goldpinger-statistics",
			},
			Name:          "kots/goldpinger",
			Selector:      []string{"app=kotsadm"},
			ContainerName: "kotsadm",
			Command:       []string{"curl"},
			Args:          []string{"http://goldpinger.kurl.svc.cluster.local:80/check_all"},
			Timeout:       "10s",
		},
	})

	return collectors
}

func makeGoldpingerAnalyzers() []*troubleshootv1beta2.Analyze {
	analyzers := []*troubleshootv1beta2.Analyze{}

	analyzers = append(analyzers, &troubleshootv1beta2.Analyze{
		TextAnalyze: &troubleshootv1beta2.TextAnalyze{
			AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
				CheckName: "Inter-pod Networking",
			},
			FileName:    "kots/goldpinger/*/kotsadm-*/goldpinger-statistics-stdout.txt",
			RegexGroups: `"OK": ?(?P<OK>\w+)`,
			Outcomes: []*troubleshootv1beta2.Outcome{
				{
					Fail: &troubleshootv1beta2.SingleOutcome{
						When:    "OK = false",
						Message: "Some nodes have pod communication issues",
					},
				},
				{
					Pass: &troubleshootv1beta2.SingleOutcome{
						Message: "Goldpinger can communicate properly",
					},
				},
			},
		},
	})

	return analyzers
}

func makeLonghornCollectors() []*troubleshootv1beta2.Collect {
	collectors := []*troubleshootv1beta2.Collect{}

	collectors = append(collectors, &troubleshootv1beta2.Collect{
		Longhorn: &troubleshootv1beta2.Longhorn{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: "longhorn",
			},
		},
	})

	return collectors
}

func makeLonghornAnalyzers() []*troubleshootv1beta2.Analyze {
	analyzers := []*troubleshootv1beta2.Analyze{}

	analyzers = append(analyzers, &troubleshootv1beta2.Analyze{
		Longhorn: &troubleshootv1beta2.LonghornAnalyze{
			AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
				CheckName: "longhorn",
			},
		},
	})

	return analyzers
}
