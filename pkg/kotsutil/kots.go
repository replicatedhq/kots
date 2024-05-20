package kotsutil

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/pkg/errors"
	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/crypto"
	registrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	kurlscheme "github.com/replicatedhq/kurlkinds/client/kurlclientset/scheme"
	kurlv1beta1 "github.com/replicatedhq/kurlkinds/pkg/apis/cluster/v1beta1"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"github.com/replicatedhq/troubleshoot/pkg/collect"
	"github.com/replicatedhq/troubleshoot/pkg/docrewrite"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"gopkg.in/yaml.v2"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	applicationv1beta1 "sigs.k8s.io/application/api/v1beta1"
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
	velerov1.AddToScheme(scheme.Scheme)
	kurlscheme.AddToScheme(scheme.Scheme)
	applicationv1beta1.AddToScheme(scheme.Scheme)
	embeddedclusterv1beta1.AddToScheme(scheme.Scheme)
}

var (
	BrandingFontFileExtensions = map[string]string{
		".woff":  "woff",
		".woff2": "woff2",
		".ttf":   "truetype",
		".otf":   "opentype",
		".eot":   "embedded-opentype",
		".svg":   "svg",
	}
)

type OverlySimpleGVK struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   OverlySimpleMetadata `yaml:"metadata"`
}

type OverlySimpleMetadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// KotsKinds are all of the special "client-side" kinds that are packaged in
// an application. These should be pointers because they are all optional.
// But a few are still expected in the code later, so we make them not pointers,
// because other codepaths expect them to be present
type KotsKinds struct {
	KotsApplication   kotsv1beta1.Application
	Application       *applicationv1beta1.Application
	V1Beta1HelmCharts *kotsv1beta1.HelmChartList
	V1Beta2HelmCharts *kotsv1beta2.HelmChartList

	Collector     *troubleshootv1beta2.Collector
	Preflight     *troubleshootv1beta2.Preflight
	Analyzer      *troubleshootv1beta2.Analyzer
	SupportBundle *troubleshootv1beta2.SupportBundle
	Redactor      *troubleshootv1beta2.Redactor
	HostPreflight *troubleshootv1beta2.HostPreflight

	Config       *kotsv1beta1.Config
	ConfigValues *kotsv1beta1.ConfigValues

	Installation kotsv1beta1.Installation
	License      *kotsv1beta1.License

	Identity       *kotsv1beta1.Identity
	IdentityConfig *kotsv1beta1.IdentityConfig

	Backup    *velerov1.Backup
	Installer *kurlv1beta1.Installer

	LintConfig *kotsv1beta1.LintConfig

	EmbeddedClusterConfig *embeddedclusterv1beta1.Config
}

func IsKotsKind(apiVersion string, kind string) bool {
	if apiVersion == "velero.io/v1" && kind == "Backup" {
		return true
	}
	if apiVersion == "kots.io/v1beta1" {
		return true
	}
	if apiVersion == "kots.io/v1beta2" {
		return true
	}
	if apiVersion == "troubleshoot.sh/v1beta2" {
		return true
	}
	if apiVersion == "troubleshoot.replicated.com/v1beta1" {
		return true
	}
	if apiVersion == "cluster.kurl.sh/v1beta1" {
		return true
	}
	if apiVersion == "kurl.sh/v1beta1" {
		return true
	}
	// In addition to kotskinds, we exclude the embedded cluster configuration.
	if apiVersion == "embeddedcluster.replicated.com/v1beta1" {
		return true
	}
	// In addition to kotskinds, we exclude the application crd for now
	if apiVersion == "app.k8s.io/v1beta1" {
		return true
	}
	return false
}

func (k *KotsKinds) EncryptConfigValues() error {
	if k.ConfigValues == nil || k.Config == nil {
		return nil
	}

	updated := map[string]kotsv1beta1.ConfigValue{}

	for name, configValue := range k.ConfigValues.Spec.Values {
		updated[name] = configValue

		if configValue.ValuePlaintext != "" {
			// ensure it's a password type
			configItemType := ""

			for _, group := range k.Config.Spec.Groups {
				for _, item := range group.Items {
					if item.Name == name {
						configItemType = item.Type
						goto Found
					}
				}
			}
		Found:

			if configItemType == "" {
				return errors.Errorf("Cannot encrypt item %q because item type was not found", name)
			}
			if configItemType != "password" {
				return errors.Errorf("Cannot encrypt item %q because item type was %q (not password)", name, configItemType)
			}

			encrypted := crypto.Encrypt([]byte(configValue.ValuePlaintext))
			encoded := base64.StdEncoding.EncodeToString(encrypted)

			configValue.Value = encoded
			configValue.ValuePlaintext = ""

			updated[name] = configValue
		}
	}

	k.ConfigValues.Spec.Values = updated

	return nil
}

func (k *KotsKinds) DecryptConfigValues() error {
	if k.ConfigValues == nil {
		return nil
	}

	updated := map[string]kotsv1beta1.ConfigValue{}

	for name, configValue := range k.ConfigValues.Spec.Values {
		// config values doesn't know the type..
		// we could look it up in the config
		// or we can just try to decode and decrypt it

		updated[name] = configValue // will be overwritten if we decrypt anything

		if configValue.Value != "" {
			decoded, err := base64.StdEncoding.DecodeString(configValue.Value)
			if err != nil {
				continue
			}
			decrypted, err := crypto.Decrypt(decoded)
			if err != nil {
				continue
			}

			configValue.Value = ""
			configValue.ValuePlaintext = string(decrypted)

			updated[name] = configValue
		}
	}

	k.ConfigValues.Spec.Values = updated

	return nil
}

func (k *KotsKinds) IsConfigurable() bool {
	if k == nil || k.Config == nil {
		return false
	}
	return len(k.Config.Spec.Groups) > 0
}

func (k *KotsKinds) HasPreflights() bool {
	if k == nil || k.Preflight == nil {
		return false
	}
	return len(k.Preflight.Spec.Analyzers) > 0
}

func (o KotsKinds) Marshal(g string, v string, k string) (string, error) {
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	if g == "kots.io" {
		if v == "v1beta1" {
			switch k {
			case "Application":
				var b bytes.Buffer
				if err := s.Encode(&o.KotsApplication, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode kots application")
				}
				return string(b.Bytes()), nil
			case "Installation":
				var b bytes.Buffer
				if err := s.Encode(&o.Installation, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode installation")
				}
				return string(b.Bytes()), nil
			case "License":
				if o.License == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.License, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode license")
				}
				return string(b.Bytes()), nil
			case "Config":
				if o.Config == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.Config, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode config")
				}
				return string(b.Bytes()), nil
			case "ConfigValues":
				if o.ConfigValues == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.ConfigValues, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode configvalues")
				}
				return string(b.Bytes()), nil
			case "Identity":
				if o.Identity == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.Identity, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode identity")
				}
				return string(b.Bytes()), nil
			case "IdentityConfig":
				if o.IdentityConfig == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.IdentityConfig, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode identityconfig")
				}
				return string(b.Bytes()), nil
			case "HelmChartList":
				if o.V1Beta1HelmCharts == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.V1Beta1HelmCharts, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode v1beta1 helmcharts")
				}
				return string(b.Bytes()), nil
			}
		} else if v == "v1beta2" {
			switch k {
			case "HelmChartList":
				if o.V1Beta2HelmCharts == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.V1Beta2HelmCharts, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode v1beta2 helmcharts")
				}
				return string(b.Bytes()), nil
			}
		}
	}

	if g == "troubleshoot.replicated.com" || g == "troubleshoot.sh" {
		if v == "v1beta1" || v == "v1beta2" {
			switch k {
			case "Collector":
				collector := o.Collector
				// SupportBundle overwrites Collector if defined
				if o.SupportBundle != nil {
					collector = SupportBundleToCollector(o.SupportBundle)
				}
				if collector == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(collector, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode collector")
				}
				return string(b.Bytes()), nil
			case "Analyzer":
				analyzer := o.Analyzer
				// SupportBundle overwrites Analyzer if defined
				if o.SupportBundle != nil {
					analyzer = SupportBundleToAnalyzer(o.SupportBundle)
				}
				if analyzer == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(analyzer, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode analyzer")
				}
				return string(b.Bytes()), nil
			case "Preflight":
				if o.Preflight == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.Preflight, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode preflight")
				}
				return string(b.Bytes()), nil
			case "HostPreflight":
				if o.HostPreflight == nil {
					return "", nil
				}

				var b bytes.Buffer
				if err := s.Encode(o.HostPreflight, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode hostpreflight")
				}
				return string(b.Bytes()), nil
			case "SupportBundle":
				if o.SupportBundle == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.SupportBundle, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode support bundle")
				}
				return string(b.Bytes()), nil
			case "Redactor":
				if o.Redactor == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.Redactor, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode redactor")
				}
				return string(b.Bytes()), nil
			}
		}
	}

	if g == "app.k8s.io" {
		if v == "v1beta1" {
			if k == "Application" {
				if o.Application == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.Application, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode application")
				}
				return string(b.Bytes()), nil
			}
		}
	}

	if g == "velero.io" {
		if v == "v1" {
			if k == "Backup" {
				if o.Backup == nil {
					if util.IsEmbeddedCluster() {
						// return the default backup object
						backup := &velerov1.Backup{
							TypeMeta: metav1.TypeMeta{
								APIVersion: "velero.io/v1",
								Kind:       "Backup",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "backup",
							},
						}
						o.Backup = backup
					} else {
						return "", nil
					}

				}
				var b bytes.Buffer
				if err := s.Encode(o.Backup, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode backup")
				}
				return string(b.Bytes()), nil
			}
		}
	}

	if g == "kurl.sh" || g == "cluster.kurl.sh" {
		if v == "v1beta1" {
			if k == "Installer" {
				if o.Installer == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.Installer, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode installer")
				}
				return string(b.Bytes()), nil
			}
		}
	}

	if g == "embeddedcluster.replicated.com" && v == "v1beta1" && k == "Config" {
		if o.EmbeddedClusterConfig == nil {
			return "", nil
		}
		var b bytes.Buffer
		if err := s.Encode(o.EmbeddedClusterConfig, &b); err != nil {
			return "", errors.Wrap(err, "failed to encode embedded cluster config")
		}
		return string(b.Bytes()), nil
	}

	return "", errors.Errorf("unknown gvk %s/%s, Kind=%s", g, v, k)
}

func (k *KotsKinds) addKotsKinds(content []byte) error {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	// kots kinds could be part of a multi-yaml doc
	docs := util.ConvertToSingleDocs(content)
	for _, doc := range docs {
		decoded, gvk, err := decode(doc, nil, nil)
		if err != nil {
			return errors.Wrapf(err, "failed to decode: %v", string(content))
		}

		if strings.HasPrefix(gvk.String(), "troubleshoot.replicated.com/v1beta1,") {
			doc, err = docrewrite.ConvertToV1Beta2(doc)
			if err != nil {
				return errors.Wrap(err, "failed to convert to v1beta2")
			}
			decoded, gvk, err = decode(doc, nil, nil)
			if err != nil {
				return errors.Wrapf(err, "failed to decode troubleshoot doc: %v", string(content))
			}
		}

		switch gvk.String() {
		case "kots.io/v1beta1, Kind=Config":
			k.Config = decoded.(*kotsv1beta1.Config)
		case "kots.io/v1beta1, Kind=ConfigValues":
			k.ConfigValues = decoded.(*kotsv1beta1.ConfigValues)
		case "kots.io/v1beta1, Kind=Application":
			k.KotsApplication = *decoded.(*kotsv1beta1.Application)
		case "kots.io/v1beta1, Kind=License":
			k.License = decoded.(*kotsv1beta1.License)
		case "kots.io/v1beta1, Kind=Identity":
			k.Identity = decoded.(*kotsv1beta1.Identity)
		case "kots.io/v1beta1, Kind=IdentityConfig":
			k.IdentityConfig = decoded.(*kotsv1beta1.IdentityConfig)
		case "kots.io/v1beta1, Kind=Installation":
			k.Installation = *decoded.(*kotsv1beta1.Installation)
		case "kots.io/v1beta1, Kind=HelmChart":
			if k.V1Beta1HelmCharts == nil {
				k.V1Beta1HelmCharts = &kotsv1beta1.HelmChartList{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "HelmChartList",
					},
				}
			}
			k.V1Beta1HelmCharts.Items = append(k.V1Beta1HelmCharts.Items, *decoded.(*kotsv1beta1.HelmChart))
		case "kots.io/v1beta2, Kind=HelmChart":
			if k.V1Beta2HelmCharts == nil {
				k.V1Beta2HelmCharts = &kotsv1beta2.HelmChartList{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta2",
						Kind:       "HelmChartList",
					},
				}
			}
			k.V1Beta2HelmCharts.Items = append(k.V1Beta2HelmCharts.Items, *decoded.(*kotsv1beta2.HelmChart))
		case "kots.io/v1beta1, Kind=LintConfig":
			k.LintConfig = decoded.(*kotsv1beta1.LintConfig)
		case "troubleshoot.sh/v1beta2, Kind=Collector":
			k.Collector = decoded.(*troubleshootv1beta2.Collector)
		case "troubleshoot.sh/v1beta2, Kind=Analyzer":
			k.Analyzer = decoded.(*troubleshootv1beta2.Analyzer)
		case "troubleshoot.sh/v1beta2, Kind=SupportBundle":
			k.SupportBundle = decoded.(*troubleshootv1beta2.SupportBundle)
		case "troubleshoot.sh/v1beta2, Kind=Redactor":
			k.Redactor = decoded.(*troubleshootv1beta2.Redactor)
		case "troubleshoot.sh/v1beta2, Kind=Preflight":
			k.Preflight = decoded.(*troubleshootv1beta2.Preflight)
		case "troubleshoot.sh/v1beta2, Kind=HostPreflight":
			k.HostPreflight = decoded.(*troubleshootv1beta2.HostPreflight)
		case "velero.io/v1, Kind=Backup":
			k.Backup = decoded.(*velerov1.Backup)
		case "kurl.sh/v1beta1, Kind=Installer", "cluster.kurl.sh/v1beta1, Kind=Installer":
			k.Installer = decoded.(*kurlv1beta1.Installer)
		case "app.k8s.io/v1beta1, Kind=Application":
			k.Application = decoded.(*applicationv1beta1.Application)
		case "embeddedcluster.replicated.com/v1beta1, Kind=Config":
			k.EmbeddedClusterConfig = decoded.(*embeddedclusterv1beta1.Config)
		}
	}

	return nil
}

func GetImagesFromKotsKinds(kotsKinds *KotsKinds, destRegistry *registrytypes.RegistryOptions) ([]string, error) {
	if kotsKinds == nil {
		return nil, nil
	}

	allImages := []string{}

	allImages = append(allImages, kotsKinds.KotsApplication.Spec.AdditionalImages...)

	collectors := make([]*troubleshootv1beta2.Collect, 0)
	if kotsKinds.SupportBundle != nil {
		collectors = append(collectors, kotsKinds.SupportBundle.Spec.Collectors...)
	}
	if kotsKinds.Collector != nil {
		collectors = append(collectors, kotsKinds.Collector.Spec.Collectors...)
	}
	if kotsKinds.Preflight != nil {
		collectors = append(collectors, kotsKinds.Preflight.Spec.Collectors...)
	}

	for _, c := range collectors {
		collector := troubleshootv1beta2.GetCollector(c)
		if collector == nil {
			continue
		}

		collectorImages := []string{}
		if imageRunner, ok := collector.(collect.ImageRunner); ok {
			collectorImages = append(collectorImages, imageRunner.GetImage())
		} else if podSpecRunner, ok := collector.(collect.PodSpecRunner); ok {
			podSpec := podSpecRunner.GetPodSpec()
			for _, container := range podSpec.InitContainers {
				collectorImages = append(collectorImages, container.Image)
			}
			for _, container := range podSpec.Containers {
				collectorImages = append(collectorImages, container.Image)
			}
		}

		for _, image := range collectorImages {
			if image == "" {
				continue
			}
			// Images that use templates like LocalImageName should be included in application's additionalImages list.
			// We want the original image names here only, not the templated ones.
			if strings.Contains(image, "repl{{") || strings.Contains(image, "{{repl") {
				continue
			}
			if destRegistry != nil {
				dockerRef, err := dockerref.ParseDockerRef(image)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse docker ref %q", image)
				}
				if strings.HasPrefix(destRegistry.Endpoint, dockerref.Domain(dockerRef)) {
					// image points to the destination registry
					continue
				}
			}
			allImages = append(allImages, image)
		}
	}

	return allImages, nil
}

// create a new kots kinds, ensuring that the require objets exist as empty defaults
func EmptyKotsKinds() KotsKinds {
	kotsKinds := KotsKinds{
		Installation: kotsv1beta1.Installation{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kots.io/v1beta1",
				Kind:       "Installation",
			},
		},
		KotsApplication: kotsv1beta1.Application{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kots.io/v1beta1",
				Kind:       "Application",
			},
		},
	}

	return kotsKinds
}

// GetKotsKindsPath returns the path to load kots kinds from in an app version archive created by kots
func GetKotsKindsPath(archive string) string {
	if archive == "" {
		return ""
	}

	kotsKindsPath := archive
	if _, err := os.Stat(filepath.Join(archive, "kotsKinds")); err == nil {
		// contains the rendered kots kinds if exists, prioritize it over upstream. only newer versions of kots create this directory.
		kotsKindsPath = filepath.Join(archive, "kotsKinds")
	} else if _, err := os.Stat(filepath.Join(archive, "upstream")); err == nil {
		// contains the non-rendered kots kinds, fallback to it if kotsKinds directory doesn't exist. this directory should always exist.
		kotsKindsPath = filepath.Join(archive, "upstream")
	}

	return kotsKindsPath
}

// LoadKotsKinds loads kots kinds from an app version archive created by kots.
// it loads the rendered kots kinds if they exist (should always be the case for app version archives created by newer kots versions).
// otherwise it loads the non-rendered kots kinds (app version archives created by older kots versions).
func LoadKotsKinds(archive string) (*KotsKinds, error) {
	return LoadKotsKindsWithOpts(archive, LoadKotsKindsOptions{Strict: true})
}

// When strict is true, it returns an error if it fails to load any kotskinds file.
// When strict is false, kotskinds with errors will be ignored if they cannot be parsed.
type LoadKotsKindsOptions struct {
	Strict bool
}

func LoadKotsKindsWithOpts(archive string, opts LoadKotsKindsOptions) (*KotsKinds, error) {
	kotsKinds := EmptyKotsKinds()

	fromDir := GetKotsKindsPath(archive)
	if fromDir == "" {
		return &kotsKinds, nil
	}

	err := filepath.Walk(fromDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				return errors.Wrapf(err, "failed to read file %s", path)
			}

			o := OverlySimpleGVK{}

			if err := yaml.Unmarshal(contents, &o); err != nil {
				// can't parse as a simple GVK, most probably not a yaml file. log an error based on extension and continue
				if ext := filepath.Ext(path); ext == ".yaml" || ext == ".yml" {
					logger.Errorf("Failed to parse yaml file %s: %v", path, err)
				}
				return nil
			}

			if !IsKotsKind(o.APIVersion, o.Kind) {
				return nil
			}

			if err := kotsKinds.addKotsKinds(contents); err != nil {
				if opts.Strict {
					return errors.Wrapf(err, "failed to add kots kinds from %s", path)
				}
			}

			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk upstream dir")
	}

	return &kotsKinds, nil
}

func KotsKindsFromMap(kotsKindsMap map[string][]byte) (*KotsKinds, error) {
	kotsKinds := EmptyKotsKinds()

	for path, content := range kotsKindsMap {
		if err := kotsKinds.addKotsKinds(content); err != nil {
			return nil, errors.Wrapf(err, "failed to parse kots kind %s", path)
		}
	}

	return &kotsKinds, nil
}

func WriteKotsKinds(kotsKinds map[string][]byte, rootDir string) error {
	// Remove any existing files from previous renders.
	_, err := os.Stat(rootDir)
	if err == nil {
		if err := os.RemoveAll(rootDir); err != nil {
			return errors.Wrap(err, "failed to remove previous content in base")
		}
	}

	if err := os.MkdirAll(rootDir, 0744); err != nil {
		return errors.Wrap(err, "failed to mkdir for base root")
	}

	for filePath, content := range kotsKinds {
		fileRenderPath := filepath.Join(rootDir, filePath)
		d, _ := path.Split(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0755); err != nil {
				return errors.Wrapf(err, "failed to mkdir %s", d)
			}
		}

		if err := os.WriteFile(fileRenderPath, content, 0644); err != nil {
			return errors.Wrap(err, "failed to write base file")
		}
	}

	return nil
}

func loadRuntimeObjectsFromPath(apiVersion, kind, fromDir string) ([]runtime.Object, error) {
	objects := []runtime.Object{}
	decode := scheme.Codecs.UniversalDeserializer().Decode
	err := filepath.Walk(fromDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				return errors.Wrap(err, "failed to read file")
			}

			if !IsApiVersionKind(contents, apiVersion, kind) {
				return nil
			}

			decoded, gvk, err := decode(contents, nil, nil)
			if err != nil {
				return errors.Wrapf(err, "failed to decode: %v", string(contents))
			}

			if gvk.String() == fmt.Sprintf("%s, Kind=%s", apiVersion, kind) {
				objects = append(objects, decoded)
			}

			return nil
		})
	if err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			return nil, errors.Wrap(err, "failed to walk upstream dir")
		}
	}

	return objects, nil
}

func IsApiVersionKind(content []byte, apiVersion, kind string) bool {
	gvk := OverlySimpleGVK{}
	if err := yaml.Unmarshal(content, &gvk); err != nil {
		return false
	}
	if gvk.APIVersion == apiVersion && gvk.Kind == kind {
		return true
	}
	return false
}

func LoadInstallationFromPath(installationFilePath string) (*kotsv1beta1.Installation, error) {
	installationData, err := os.ReadFile(installationFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read installation file")
	}

	return LoadInstallationFromContents(installationData)
}

func LoadSupportBundleFromContents(data []byte) (*troubleshootv1beta2.SupportBundle, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(data), nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode: %v", string(data))
	}

	if gvk.Group != "troubleshoot.sh" || gvk.Version != "v1beta2" || gvk.Kind != "SupportBundle" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*troubleshootv1beta2.SupportBundle), nil
}

func FindKotsAppInPath(fromDir string) (*kotsv1beta1.Application, error) {
	objects, err := loadRuntimeObjectsFromPath("kots.io/v1beta1", "Application", fromDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load v1beta2 HelmCharts from %s", fromDir)
	}

	if len(objects) == 0 {
		return nil, nil
	}

	// we only support having one kots app spec
	return objects[0].(*kotsv1beta1.Application), nil
}

func LoadKotsAppFromContents(data []byte) (*kotsv1beta1.Application, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(data), nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode: %v", string(data))
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Application" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.Application), nil
}

func LoadV1Beta1HelmChartListFromContents(data []byte) (*kotsv1beta1.HelmChartList, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(data, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode: %v", string(data))
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "HelmChartList" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.HelmChartList), nil
}

func LoadV1Beta1HelmChartFromContents(content []byte) (*kotsv1beta1.HelmChart, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode: %v", string(content))
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "HelmChart" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.HelmChart), nil
}

func LoadV1Beta2HelmChartFromContents(content []byte) (*kotsv1beta2.HelmChart, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode: %v", string(content))
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta2" || gvk.Kind != "HelmChart" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta2.HelmChart), nil
}

func LoadInstallationFromContents(installationData []byte) (*kotsv1beta1.Installation, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(installationData), nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode: %v", string(installationData))
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Installation" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.Installation), nil
}

func LoadLicenseFromPath(licenseFilePath string) (*kotsv1beta1.License, error) {
	licenseData, err := os.ReadFile(licenseFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read license file")
	}

	return LoadLicenseFromBytes(licenseData)
}

func LoadLicenseFromBytes(data []byte) (*kotsv1beta1.License, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(data), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode license data")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "License" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.License), nil
}

func LoadEmbeddedClusterConfigFromBytes(data []byte) (*embeddedclusterv1beta1.Config, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(data), nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode: %v", string(data))
	}
	if gvk.Group != "embeddedcluster.replicated.com" || gvk.Version != "v1beta1" || gvk.Kind != "Config" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}
	return obj.(*embeddedclusterv1beta1.Config), nil
}

func LoadConfigValuesFromFile(configValuesFilePath string) (*kotsv1beta1.ConfigValues, error) {
	configValuesData, err := os.ReadFile(configValuesFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read configvalues file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(configValuesData), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode configvalues data")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "ConfigValues" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.ConfigValues), nil
}

func FindConfigInPath(fromDir string) (*kotsv1beta1.Config, error) {
	objects, err := loadRuntimeObjectsFromPath("kots.io/v1beta1", "Config", fromDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load Config from %s", fromDir)
	}

	if len(objects) == 0 {
		return nil, nil
	}

	// we only support having one config spec
	return objects[0].(*kotsv1beta1.Config), nil
}

func LoadConfigFromBytes(data []byte) (*kotsv1beta1.Config, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(data, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode config data")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Config" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.Config), nil
}

func LoadConfigValuesFromBytes(data []byte) (*kotsv1beta1.ConfigValues, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(data, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode config data")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "ConfigValues" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.ConfigValues), nil
}

func LoadPreflightFromContents(content []byte) (*troubleshootv1beta2.Preflight, error) {
	content, err := docrewrite.ConvertToV1Beta2(content)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert to v1beta2")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "not a preflight")
	}

	if gvk.String() == "troubleshoot.sh/v1beta2, Kind=Preflight" {
		return obj.(*troubleshootv1beta2.Preflight), nil
	}

	if gvk.String() == "troubleshoot.replicated.com/v1beta1, Kind=Preflight" {
		return obj.(*troubleshootv1beta2.Preflight), nil
	}

	return nil, errors.Errorf("not a preflight: %s", gvk.String())

}

func LoadBackupFromContents(content []byte) (*velerov1.Backup, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode: %v", string(content))
	}

	if gvk.String() != "velero.io/v1, Kind=Backup" {
		return nil, errors.Errorf("unexpected gvk: %s", gvk.String())
	}

	return obj.(*velerov1.Backup), nil
}

func LoadApplicationFromContents(content []byte) (*applicationv1beta1.Application, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode: %v", string(content))
	}

	if gvk.String() != "app.k8s.io/v1beta1, Kind=Application" {
		return nil, errors.Errorf("unexpected gvk: %s", gvk.String())
	}

	return obj.(*applicationv1beta1.Application), nil
}

func LoadApplicationFromBytes(content []byte) (*kotsv1beta1.Application, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode: %v", string(content))
	}

	if gvk.String() != "kots.io/v1beta1, Kind=Application" {
		return nil, errors.Errorf("unexpected gvk: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.Application), nil
}

func SupportBundleToCollector(sb *troubleshootv1beta2.SupportBundle) *troubleshootv1beta2.Collector {
	return &troubleshootv1beta2.Collector{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "troubleshoot.sh/v1beta2",
			Kind:       "Collector",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-collector", sb.Name),
		},
		Spec: troubleshootv1beta2.CollectorSpec{
			Collectors: sb.Spec.Collectors,
		},
	}
}

func SupportBundleToAnalyzer(sb *troubleshootv1beta2.SupportBundle) *troubleshootv1beta2.Analyzer {
	return &troubleshootv1beta2.Analyzer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "troubleshoot.sh/v1beta2",
			Kind:       "Analyzer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-analyzer", sb.Name),
		},
		Spec: troubleshootv1beta2.AnalyzerSpec{
			Analyzers: sb.Spec.Analyzers,
		},
	}
}

type InstallationParams struct {
	KotsadmRegistry        string
	SkipImagePush          bool
	SkipPreflights         bool
	SkipCompatibilityCheck bool
	RegistryIsReadOnly     bool
	EnableImageDeletion    bool
	EnsureRBAC             bool
	SkipRBACCheck          bool
	UseMinimalRBAC         bool
	StrictSecurityContext  bool
	WaitDuration           time.Duration
	WithMinio              bool
	AppVersionLabel        string
}

func GetInstallationParams(configMapName string) (InstallationParams, error) {
	autoConfig := InstallationParams{}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return autoConfig, errors.Wrap(err, "failed to get k8s clientset")
	}

	isKurl, err := kurl.IsKurl(clientset)
	if err != nil {
		return autoConfig, errors.Wrap(err, "failed to check if cluster is kurl")
	}

	kotsadmConfigMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return autoConfig, nil
		}
		return autoConfig, errors.Wrap(err, "failed to get existing kotsadm config map")
	}

	autoConfig.KotsadmRegistry = kotsadmConfigMap.Data["kotsadm-registry"]
	autoConfig.SkipImagePush, _ = strconv.ParseBool(kotsadmConfigMap.Data["initial-app-images-pushed"])
	autoConfig.SkipPreflights, _ = strconv.ParseBool(kotsadmConfigMap.Data["skip-preflights"])
	autoConfig.SkipCompatibilityCheck, _ = strconv.ParseBool(kotsadmConfigMap.Data["skip-compatibility-check"])
	autoConfig.RegistryIsReadOnly, _ = strconv.ParseBool(kotsadmConfigMap.Data["registry-is-read-only"])
	autoConfig.EnsureRBAC, _ = strconv.ParseBool(kotsadmConfigMap.Data["ensure-rbac"])
	autoConfig.SkipRBACCheck, _ = strconv.ParseBool(kotsadmConfigMap.Data["skip-rbac-check"])
	autoConfig.UseMinimalRBAC, _ = strconv.ParseBool(kotsadmConfigMap.Data["use-minimal-rbac"])
	autoConfig.StrictSecurityContext, _ = strconv.ParseBool(kotsadmConfigMap.Data["strict-security-context"])
	autoConfig.WaitDuration, _ = time.ParseDuration(kotsadmConfigMap.Data["wait-duration"])
	autoConfig.WithMinio, _ = strconv.ParseBool(kotsadmConfigMap.Data["with-minio"])
	autoConfig.AppVersionLabel = kotsadmConfigMap.Data["app-version-label"]

	if enableImageDeletion, ok := kotsadmConfigMap.Data["enable-image-deletion"]; ok {
		autoConfig.EnableImageDeletion, _ = strconv.ParseBool(enableImageDeletion)
	} else {
		autoConfig.EnableImageDeletion = isKurl
	}

	return autoConfig, nil
}

func LoadIngressConfigFromContents(content []byte) (*kotsv1beta1.IngressConfig, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode content")
	}

	if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "IngressConfig" {
		return obj.(*kotsv1beta1.IngressConfig), nil
	}

	return nil, errors.Errorf("unexpected gvk: %s", gvk.String())
}

func LoadIdentityFromContents(content []byte) (*kotsv1beta1.Identity, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode content")
	}

	if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Identity" {
		return obj.(*kotsv1beta1.Identity), nil
	}

	return nil, errors.Errorf("unexpected gvk: %s", gvk.String())
}

func LoadIdentityConfigFromContents(content []byte) (*kotsv1beta1.IdentityConfig, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode content")
	}

	if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "IdentityConfig" {
		return obj.(*kotsv1beta1.IdentityConfig), nil
	}

	return nil, errors.Errorf("unexpected gvk: %s", gvk.String())
}

func EncodeIngressConfig(ingressConfig kotsv1beta1.IngressConfig) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	err := s.Encode(&ingressConfig, buf)
	return buf.Bytes(), err
}

func EncodeIdentityConfig(spec kotsv1beta1.IdentityConfig) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	err := s.Encode(&spec, buf)
	return buf.Bytes(), err
}

func IsKotsVersionCompatibleWithApp(kotsApplication kotsv1beta1.Application, isInstall bool) bool {
	actualSemver, err := semver.ParseTolerant(buildversion.Version())
	if err != nil {
		return true
	}

	if kotsApplication.Spec.MinKotsVersion != "" {
		minSemver, err := semver.ParseTolerant(kotsApplication.Spec.MinKotsVersion)
		if err != nil {
			logger.Error(errors.Wrap(err, "minimum kots version specified in the application spec is invalid"))
		} else if actualSemver.LT(minSemver) {
			return false
		}
	}

	if isInstall && kotsApplication.Spec.TargetKotsVersion != "" {
		targetSemver, err := semver.ParseTolerant(kotsApplication.Spec.TargetKotsVersion)
		if err != nil {
			logger.Error(errors.Wrap(err, "target kots version specified in the application spec is invalid"))
		} else if actualSemver.GT(targetSemver) {
			return false
		}
	}

	return true
}

func GetIncompatbileKotsVersionMessage(kotsApplication kotsv1beta1.Application, isInstall bool) string {
	appName := kotsApplication.Spec.Title
	if appName == "" {
		appName = "the app"
	}

	desiredKotsVersion := kotsApplication.Spec.TargetKotsVersion
	if desiredKotsVersion == "" {
		desiredKotsVersion = kotsApplication.Spec.MinKotsVersion
	}

	if isInstall {
		return fmt.Sprintf(
			"This version of %s requires a different version of KOTS from what you currently have installed.\nInstall KOTS version %s and try again.",
			appName,
			desiredKotsVersion,
		)
	}

	return fmt.Sprintf(
		"This version of %s requires a version of KOTS that is different than what you currently have installed. Upgrade KOTS to version %s, and then download this application version again in the admin console or with the 'kots upstream download' command.",
		appName,
		desiredKotsVersion,
	)
}

func IsKotsAutoUpgradeSupported(app *kotsv1beta1.Application) bool {
	if app == nil {
		return false
	}

	for _, f := range app.Spec.ConsoleFeatureFlags {
		if f == "admin-console-auto-updates" {
			return true
		}
	}

	return false
}

func RemoveAppVersionLabelFromInstallationParams(configMapName string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	kotsadmConfigMap, err := clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get existing kotsadm config map")
	}

	if kotsadmConfigMap.Data["app-version-label"] == "" {
		return nil
	}

	delete(kotsadmConfigMap.Data, "app-version-label")

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.TODO(), kotsadmConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update kotsadm config map")
	}

	return nil
}

func FindAirgapMetaInDir(root string) (*kotsv1beta1.Airgap, error) {
	files, err := os.ReadDir(root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read airgap directory content")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		contents, err := os.ReadFile(filepath.Join(root, file.Name()))
		if err != nil {
			// TODO: log?
			continue
		}

		airgap, err := LoadAirgapFromBytes(contents)
		if err != nil {
			// TODO: log?
			continue
		}

		return airgap, nil
	}

	return nil, errors.Errorf("airgap meta not found in %s", root)
}

func FindAirgapMetaInBundle(airgapBundle string) (*kotsv1beta1.Airgap, error) {
	content, err := archives.GetFileFromAirgap("airgap.yaml", airgapBundle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract airgap.yaml file")
	}
	return LoadAirgapFromBytes(content)
}

func LoadAirgapFromBytes(data []byte) (*kotsv1beta1.Airgap, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(data), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode airgap data")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Airgap" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.Airgap), nil
}

func GetKOTSBinPath() string {
	if util.PodNamespace != "" {
		// we're inside the kotsadm pod, the kots binary exists at /kots
		return "/kots"
	} else {
		// we're not inside the kotsadm pod, return the command used to run kots
		return os.Args[0]
	}
}

func LoadBrandingArchiveFromPath(archivePath string) (*bytes.Buffer, error) {
	fileInfo, err := os.Stat(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to stat branding archive path")
	}
	if !fileInfo.IsDir() {
		return nil, errors.New("branding archive path is not a directory")
	}

	buf := bytes.NewBuffer(nil)
	gz := gzip.NewWriter(buf)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	hasFiles := false

	err = filepath.Walk(archivePath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			ext := filepath.Ext(path)
			_, isFontFile := BrandingFontFileExtensions[ext]
			if ext != ".yaml" && ext != ".css" && !isFontFile {
				return nil
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				return errors.Wrap(err, "failed to read file")
			}

			name := strings.TrimPrefix(path, archivePath+string(os.PathSeparator))

			if ext == ".yaml" {
				_, gvk, err := scheme.Codecs.UniversalDeserializer().Decode(contents, nil, nil)
				if err != nil {
					return nil
				}

				if gvk.String() != "kots.io/v1beta1, Kind=Application" {
					return nil
				}

				name = "application.yaml"
			}

			hdr := &tar.Header{
				Name:    name,
				Mode:    int64(info.Mode()),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return errors.Wrap(err, "failed to write tar header")
			}

			if _, err := tw.Write(contents); err != nil {
				return errors.Wrap(err, "failed to write tar contents")
			}

			hasFiles = true

			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk archive path")
	}

	if !hasFiles {
		return bytes.NewBuffer(nil), nil
	}

	return buf, nil
}

// FilterV1Beta1ChartsWithV1Beta2Charts filters out any v1beta1 charts where a cooresponding v1beta2 chart exists with the same release name
// (or chart name if release name is not set)
func FilterV1Beta1ChartsWithV1Beta2Charts(v1Beta1Charts []kotsv1beta1.HelmChart, v1Beta2Charts []kotsv1beta2.HelmChart) []kotsv1beta1.HelmChart {
	v1Beta2ChartMap := map[string]bool{}
	for _, v1beta2Chart := range v1Beta2Charts {
		v1Beta2ChartMap[v1beta2Chart.GetReleaseName()] = true
	}

	filteredCharts := []kotsv1beta1.HelmChart{}
	for _, chart := range v1Beta1Charts {
		if v1Beta2ChartMap[chart.GetReleaseName()] {
			continue
		}
		filteredCharts = append(filteredCharts, chart)
	}

	return filteredCharts
}

func MustMarshalInstallation(installation *kotsv1beta1.Installation) []byte {
	b, err := MarshalRuntimeObject(installation)
	if err != nil {
		panic(err)
	}
	return b
}

func MarshalRuntimeObject(obj runtime.Object) ([]byte, error) {
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(obj, &b); err != nil {
		return nil, errors.Wrap(err, "failed to encode object")
	}

	return b.Bytes(), nil
}

// this is here to avoid a circular dependency
func SaveInstallation(installation *kotsv1beta1.Installation, upstreamDir string) error {
	filename := path.Join(upstreamDir, "userdata", "installation.yaml")
	err := os.WriteFile(filename, MustMarshalInstallation(installation), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write installation")
	}
	return nil
}

// TODO NOW: download via replicated.app
func DownloadKOTSBinary(version string) (string, error) {
	url := fmt.Sprintf("https://github.com/replicatedhq/kots/releases/download/%s/kots_linux_amd64.tar.gz", version)
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "failed to get")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("unexpected status code %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "kots")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file")
	}

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to get new gzip reader")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", errors.Wrap(err, "failed to get read archive")
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}
		if header.Name != "kots" {
			continue
		}

		if _, err := io.Copy(tmpFile, tarReader); err != nil {
			return "", errors.Wrap(err, "failed to copy kots binary")
		}
		break
	}

	return "", errors.New("kots binary not found in archive")
}
