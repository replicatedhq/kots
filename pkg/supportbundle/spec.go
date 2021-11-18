package supportbundle

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotstypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/registry"
	"github.com/replicatedhq/kots/pkg/render/helper"
	"github.com/replicatedhq/kots/pkg/snapshot"
	kotssnapshot "github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle/defaultspec"
	"github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/kots/pkg/util"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"go.uber.org/multierr"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

// CreateRenderedSpec creates the support bundle specification from defaults and the kots app
func CreateRenderedSpec(appID string, sequence int64, kotsKinds *kotsutil.KotsKinds, opts types.TroubleshootOptions) (*troubleshootv1beta2.SupportBundle, error) {
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

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	minimalRBACNamespaces := []string{}
	if !k8sutil.IsKotsadmClusterScoped(context.TODO(), clientset, util.PodNamespace) {
		minimalRBACNamespaces = append(minimalRBACNamespaces, util.PodNamespace)
		minimalRBACNamespaces = append(minimalRBACNamespaces, kotsKinds.KotsApplication.Spec.AdditionalNamespaces...)
		requiresAccess, err := kotssnapshot.CheckKotsadmVeleroAccess(context.TODO(), util.PodNamespace)
		if err != nil {
			logger.Errorf("Failed to check kotsadm velero access for the support bundle: %v", err)
		} else if !requiresAccess {
			veleroNamespace, err := kotssnapshot.DetectVeleroNamespace(context.TODO(), clientset, util.PodNamespace)
			if err != nil {
				logger.Errorf("Failed to detect velero namespace for the support bundle: %v", err)
			} else {
				minimalRBACNamespaces = append(minimalRBACNamespaces, veleroNamespace)
			}
		}
	}

	app, err := store.GetStore().GetApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}

	builtBundle, err = injectDefaults(app, builtBundle, opts, minimalRBACNamespaces)
	if err != nil {
		return nil, errors.Wrap(err, "failed to inject defaults")
	}

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(builtBundle, &b); err != nil {
		return nil, errors.Wrap(err, "failed to encode support bundle")
	}

	templatedSpec := b.Bytes()

	renderedSpec, err := helper.RenderAppFile(app, &sequence, templatedSpec, kotsKinds, util.PodNamespace)
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

	secretName := GetSpecSecretName(app.Slug)
	existingSecret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), secretName, metav1.GetOptions{})
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
				Namespace: util.PodNamespace,
				Labels:    kotstypes.GetKotsadmLabels(),
			},
			Data: map[string][]byte{
				SpecDataKey: renderedSpec,
			},
		}

		_, err = clientset.CoreV1().Secrets(util.PodNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
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

	_, err = clientset.CoreV1().Secrets(util.PodNamespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to update support bundle secret")
	}

	return supportBundle, nil
}

// injectDefaults injects the kotsadm default collectors/analyzers in the the support bundle specification.
func injectDefaults(app *apptypes.App, b *troubleshootv1beta2.SupportBundle, opts types.TroubleshootOptions, minimalRBACNamespaces []string) (*troubleshootv1beta2.SupportBundle, error) {
	supportBundle := b.DeepCopy()

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Errorf("Failed to get kubernetes clientset: %v", err)
	}

	var imageName string
	var pullSecret *troubleshootv1beta2.ImagePullSecrets
	if clientset != nil {
		imageName, pullSecret, err = getImageAndSecret(context.TODO(), clientset)
		if err != nil {
			logger.Errorf("Failed to get kotsadm image and secret: %v", err)
		}
	}

	if supportBundle == nil {
		supportBundle = &troubleshootv1beta2.SupportBundle{}
	}
	if supportBundle.Spec.Collectors == nil {
		supportBundle.Spec.Collectors = make([]*troubleshootv1beta2.Collect, 0)
	}
	if supportBundle.Spec.Analyzers == nil {
		supportBundle.Spec.Analyzers = make([]*troubleshootv1beta2.Analyze, 0)
	}

	supportBundle = addDefaultTroubleshoot(supportBundle, imageName, pullSecret)
	supportBundle = addDefaultDynamicTroubleshoot(supportBundle, app, imageName, pullSecret)
	supportBundle = populateNamespaces(supportBundle, minimalRBACNamespaces)
	supportBundle = deduplicatedCollectors(supportBundle)
	supportBundle = deduplicatedAnalyzers(supportBundle)

	// determine an upload URL
	var uploadURL string
	var redactURL string
	randomBundleID := strings.ToLower(rand.String(32))
	if opts.DisableUpload {
		//Just use the library internally
		return supportBundle, nil
	} else if opts.Origin != "" {
		uploadURL = fmt.Sprintf("%s/api/v1/troubleshoot/%s/%s", opts.Origin, app.ID, randomBundleID)
		redactURL = fmt.Sprintf("%s/api/v1/troubleshoot/supportbundle/%s/redactions", opts.Origin, randomBundleID)
	} else if opts.InCluster {
		uploadURL = fmt.Sprintf("%s/api/v1/troubleshoot/%s/%s", fmt.Sprintf("http://kotsadm.%s.svc.cluster.local:3000", util.PodNamespace), app.ID, randomBundleID)
		redactURL = fmt.Sprintf("%s/api/v1/troubleshoot/supportbundle/%s/redactions", fmt.Sprintf("http://kotsadm.%s.svc.cluster.local:3000", util.PodNamespace), randomBundleID)
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

	return supportBundle, nil
}

// if a namespace is not set for a secret/run/logs/exec/copy collector, set it to the current namespace
// if kotsadm is running with minimal rbac priviliges, only collect resources from the specified minimal rbac namespaces
func populateNamespaces(supportBundle *troubleshootv1beta2.SupportBundle, minimalRBACNamespaces []string) *troubleshootv1beta2.SupportBundle {
	next := supportBundle.DeepCopy()

	// collectors
	var collects []*troubleshootv1beta2.Collect
	for _, collect := range next.Spec.Collectors {
		if collect.Secret != nil && collect.Secret.Namespace == "" {
			collect.Secret.Namespace = util.PodNamespace
		}
		if collect.ConfigMap != nil && collect.ConfigMap.Namespace == "" {
			collect.ConfigMap.Namespace = util.PodNamespace
		}
		if collect.Run != nil && collect.Run.Namespace == "" {
			collect.Run.Namespace = util.PodNamespace
		}
		if collect.Logs != nil && collect.Logs.Namespace == "" {
			collect.Logs.Namespace = util.PodNamespace
		}
		if collect.Exec != nil && collect.Exec.Namespace == "" {
			collect.Exec.Namespace = util.PodNamespace
		}
		if collect.Copy != nil && collect.Copy.Namespace == "" {
			collect.Copy.Namespace = util.PodNamespace
		}
		if len(minimalRBACNamespaces) > 0 {
			if collect.ClusterResources != nil && len(collect.ClusterResources.Namespaces) == 0 {
				collect.ClusterResources.Namespaces = minimalRBACNamespaces
			}
		}
		collects = append(collects, collect)
	}
	next.Spec.Collectors = collects

	// analyzers
	var analyzers []*troubleshootv1beta2.Analyze
	for _, analyzer := range next.Spec.Analyzers {
		if len(minimalRBACNamespaces) > 0 {
			if analyzer.ClusterPodStatuses != nil && len(analyzer.ClusterPodStatuses.Namespaces) == 0 {
				analyzer.ClusterPodStatuses.Namespaces = minimalRBACNamespaces
			}
		}
		analyzers = append(analyzers, analyzer)
	}
	next.Spec.Analyzers = analyzers

	return next
}

func deduplicatedCollectors(supportBundle *troubleshootv1beta2.SupportBundle) *troubleshootv1beta2.SupportBundle {
	next := supportBundle.DeepCopy()

	collectors := []*troubleshootv1beta2.Collect{}

	hasClusterResources := false
	hasClusterInfo := false
	hasCeph := false
	hasLonghorn := false
	hasSysctl := false

	for _, c := range next.Spec.Collectors {
		if c.ClusterResources != nil {
			if hasClusterResources {
				continue
			}
			hasClusterResources = true
		}

		if c.ClusterInfo != nil {
			if hasClusterInfo {
				continue
			}
			hasClusterInfo = true
		}

		if c.Ceph != nil {
			if hasCeph {
				continue
			}
			hasCeph = true
		}

		if c.Longhorn != nil {
			if hasLonghorn {
				continue
			}
			hasLonghorn = true
		}

		if c.Sysctl != nil {
			if hasSysctl {
				continue
			}
			hasSysctl = true
		}

		collectors = append(collectors, c)
	}

	next.Spec.Collectors = collectors

	return next
}

func deduplicatedAnalyzers(supportBundle *troubleshootv1beta2.SupportBundle) *troubleshootv1beta2.SupportBundle {
	next := supportBundle.DeepCopy()

	analyzers := []*troubleshootv1beta2.Analyze{}

	hasClusterVersion := false
	hasLonghorn := false
	hasWeaveReport := false

	for _, a := range next.Spec.Analyzers {
		if a.ClusterVersion != nil {
			if hasClusterVersion {
				continue
			}
			hasClusterVersion = true
		}

		if a.Longhorn != nil {
			if hasLonghorn {
				continue
			}
			hasLonghorn = true
		}

		if a.WeaveReport != nil {
			if hasWeaveReport {
				continue
			}
			hasWeaveReport = true
		}

		analyzers = append(analyzers, a)
	}

	next.Spec.Analyzers = analyzers

	return next
}

// addDefaultTroubleshoot adds kots.io (github.com/replicatedhq/kots/support-bundle/spec.yaml) spec to the support bundle.
func addDefaultTroubleshoot(supportBundle *troubleshootv1beta2.SupportBundle, imageName string, pullSecret *troubleshootv1beta2.ImagePullSecrets) *troubleshootv1beta2.SupportBundle {
	next := supportBundle.DeepCopy()
	next.Spec.Collectors = append(next.Spec.Collectors, getDefaultCollectors(imageName, pullSecret)...)
	next.Spec.Analyzers = append(next.Spec.Analyzers, getDefaultAnalyzers()...)
	return next
}

func getDefaultCollectors(imageName string, pullSecret *troubleshootv1beta2.ImagePullSecrets) []*troubleshootv1beta2.Collect {
	supportBundle := defaultspec.Get()
	if imageName != "" {
		supportBundle = *populateImages(&supportBundle, imageName, pullSecret)
	}
	return supportBundle.Spec.Collectors
}

func getDefaultAnalyzers() []*troubleshootv1beta2.Analyze {
	return defaultspec.Get().Spec.Analyzers
}

// addDefaultDynamicTroubleshoot adds dynamic spec to the support bundle.
// prefer addDefaultTroubleshoot unless absolutely necessary to encourage consistency across built-in and kots.io specs.
func addDefaultDynamicTroubleshoot(supportBundle *troubleshootv1beta2.SupportBundle, app *apptypes.App, imageName string, pullSecret *troubleshootv1beta2.ImagePullSecrets) *troubleshootv1beta2.SupportBundle {
	next := supportBundle.DeepCopy()
	next.Spec.Collectors = append(next.Spec.Collectors, getDefaultDynamicCollectors(app, imageName, pullSecret)...)
	next.Spec.Analyzers = append(next.Spec.Analyzers, getDefaultDynamicAnalyzers(app)...)
	return next
}

func getDefaultDynamicCollectors(app *apptypes.App, imageName string, pullSecret *troubleshootv1beta2.ImagePullSecrets) []*troubleshootv1beta2.Collect {
	collectors := make([]*troubleshootv1beta2.Collect, 0)

	license, err := store.GetStore().GetLatestLicenseForApp(app.ID)
	if err != nil {
		logger.Errorf("Failed to load license data: %v", err)
	}

	if license != nil {
		licenseData, err := yaml.Marshal(license)
		if err != nil {
			logger.Errorf("Failed to marshal license: %v", err)
		}
		collectors = append(collectors, &troubleshootv1beta2.Collect{
			Data: &troubleshootv1beta2.Data{
				CollectorMeta: troubleshootv1beta2.CollectorMeta{
					CollectorName: "license.yaml",
				},
				Name: "kots/admin-console",
				Data: string(licenseData),
			},
		})
	}

	collectors = append(collectors, &troubleshootv1beta2.Collect{
		Data: &troubleshootv1beta2.Data{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: "namespace.txt",
			},
			Name: "kots/admin-console",
			Data: util.PodNamespace,
		},
	})

	collectors = append(collectors, &troubleshootv1beta2.Collect{
		Secret: &troubleshootv1beta2.Secret{
			CollectorMeta: troubleshootv1beta2.CollectorMeta{
				CollectorName: fmt.Sprintf("%s-registry", app.Slug),
			},
			Name:         fmt.Sprintf("%s-registry", app.Slug),
			Namespace:    util.PodNamespace,
			Key:          ".dockerconfigjson",
			IncludeValue: false,
		},
	})

	collectors = append(collectors, makeVeleroCollectors()...)

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
		appVersionArchiveCollectors, err := makeAppVersionArchiveCollectors(apps)
		if err != nil {
			logger.Errorf("Failed to make app version archive collectors: %v", err)
		}
		collectors = append(collectors, appVersionArchiveCollectors...)
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Errorf("Failed to get clientset for dynamic kurl collectors: %v", err)
	} else if kotsutil.IsKurl(clientset) {
		collectors = append(collectors, &troubleshootv1beta2.Collect{
			Sysctl: &troubleshootv1beta2.Sysctl{
				Image:           imageName,
				ImagePullSecret: pullSecret,
			},
		})
	}

	return collectors
}

func getDefaultDynamicAnalyzers(app *apptypes.App) []*troubleshootv1beta2.Analyze {
	analyzers := make([]*troubleshootv1beta2.Analyze, 0)
	analyzers = append(analyzers, makeAPIReplicaAnalyzer())

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Errorf("Failed to get clientset for dynamic kurl analyzers: %v", err)
	} else if kotsutil.IsKurl(clientset) {
		analyzers = append(analyzers,
			&troubleshootv1beta2.Analyze{
				Sysctl: &troubleshootv1beta2.SysctlAnalyze{
					AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
						CheckName: "IP forwarding not enabled",
					},
					Outcomes: []*troubleshootv1beta2.Outcome{
						{
							Fail: &troubleshootv1beta2.SingleOutcome{
								When:    "net.ipv4.ip_forward = 0",
								Message: "IP forwarding not enabled",
							},
						},
					},
				},
			},
			&troubleshootv1beta2.Analyze{
				Sysctl: &troubleshootv1beta2.SysctlAnalyze{
					AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
						CheckName: "Bridge iptables integration",
					},
					Outcomes: []*troubleshootv1beta2.Outcome{
						{
							Fail: &troubleshootv1beta2.SingleOutcome{
								When:    "net.bridge.bridge-nf-call-iptables = 0",
								Message: "Packets traversing bridge interfaces not processed by iptables",
							},
						},
					},
				},
			},
		)
	}

	return analyzers
}

func makeVeleroCollectors() []*troubleshootv1beta2.Collect {
	collectors := []*troubleshootv1beta2.Collect{}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		return collectors
	}

	veleroNamespace, err := snapshot.DetectVeleroNamespace(context.TODO(), clientset, util.PodNamespace)
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

	latestVersion, err := store.GetStore().GetLatestAppVersion(app.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest app version")
	}

	tempPath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tempPath)

	err = store.GetStore().GetAppVersionArchive(app.ID, latestVersion.Sequence, tempPath)
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
			Namespace:     util.PodNamespace,
			ContainerName: "kotsadm", // can we assume this? kotsadm-api
			ContainerPath: fileName,
			Name:          fmt.Sprintf("kots/admin-console/app/%s", app.Slug),
		},
	}, nil
}

func makeAPIReplicaAnalyzer() *troubleshootv1beta2.Analyze {
	if os.Getenv("POD_OWNER_KIND") == "deployment" {
		return &troubleshootv1beta2.Analyze{
			DeploymentStatus: &troubleshootv1beta2.DeploymentStatus{
				Name:      "kotsadm",
				Namespace: util.PodNamespace,
				Outcomes: []*troubleshootv1beta2.Outcome{
					{
						Pass: &troubleshootv1beta2.SingleOutcome{
							When:    "> 0",
							Message: "At least 1 replica of the Admin Console API is running and ready",
						},
					},
					{
						Fail: &troubleshootv1beta2.SingleOutcome{
							When:    "= 0",
							Message: "There are no replicas of the Admin Console API running and ready",
						},
					},
				},
			},
		}
	}
	return &troubleshootv1beta2.Analyze{
		StatefulsetStatus: &troubleshootv1beta2.StatefulsetStatus{
			Name:      "kotsadm",
			Namespace: util.PodNamespace,
			Outcomes: []*troubleshootv1beta2.Outcome{
				{
					Pass: &troubleshootv1beta2.SingleOutcome{
						When:    "> 0",
						Message: "At least 1 replica of the Admin Console API is running and ready",
					},
				},
				{
					Fail: &troubleshootv1beta2.SingleOutcome{
						When:    "= 0",
						Message: "There are no replicas of the Admin Console API running and ready",
					},
				},
			},
		},
	}
}

func getImageAndSecret(ctx context.Context, clientset kubernetes.Interface) (imageName string, pullSecret *troubleshootv1beta2.ImagePullSecrets, err error) {
	namespace := util.PodNamespace

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

func populateImages(supportBundle *troubleshootv1beta2.SupportBundle, imageName string, pullSecret *troubleshootv1beta2.ImagePullSecrets) *troubleshootv1beta2.SupportBundle {
	next := supportBundle.DeepCopy()

	collects := []*troubleshootv1beta2.Collect{}
	for _, collect := range next.Spec.Collectors {
		if collect.Collectd != nil && collect.Collectd.Image == "alpine" { // TODO: is this too strong of an assumption?
			collect.Collectd.Image = imageName
			collect.Collectd.ImagePullSecret = pullSecret
		}
		if collect.CopyFromHost != nil && collect.CopyFromHost.Image == "alpine" { // TODO: is this too strong of an assumption?
			collect.CopyFromHost.Image = imageName
			collect.CopyFromHost.ImagePullSecret = pullSecret
		}
		collects = append(collects, collect)
	}
	next.Spec.Collectors = collects

	return next
}
