package types

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/binaries"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/template"
	kurlv1beta1 "github.com/replicatedhq/kurl/kurlkinds/pkg/apis/cluster/v1beta1"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/collect"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	applicationv1beta1 "sigs.k8s.io/application/api/v1beta1"
)

type OverlySimpleGVK struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

type LoadKotsKindsOptions struct {
	FromDir          string
	RegistrySettings registrytypes.RegistrySettings
	AppSlug          string
	Sequence         int64
	IsAirgap         bool
	Namespace        string
}

type BuilderOptions struct {
	Config           *kotsv1beta1.Config
	ConfigValues     *kotsv1beta1.ConfigValues
	Installation     kotsv1beta1.Installation
	KotsApplication  kotsv1beta1.Application
	License          *kotsv1beta1.License
	IdentityConfig   *kotsv1beta1.IdentityConfig
	RegistrySettings registrytypes.RegistrySettings
	AppSlug          string
	Sequence         int64
	IsAirgap         bool
	Namespace        string
}

type CreateConfigValuesOptions struct {
	ApplicationName      string
	Config               *kotsv1beta1.Config
	ExistingConfigValues *kotsv1beta1.ConfigValues
	License              *kotsv1beta1.License
	App                  *kotsv1beta1.Application
	AppInfo              *template.ApplicationInfo
	VersionInfo          *template.VersionInfo
	LocalRegistry        template.LocalRegistry
	IdentityConfig       *kotsv1beta1.IdentityConfig
}

type CreateInstallationOptions struct {
	PrevInstallation     *kotsv1beta1.Installation
	PreserveInstallation bool
	ChannelID            string
	ChannelName          string
	Name                 string
	UpdateCursor         string
	VersionLabel         string
	IsRequired           bool
	ReleaseNotes         string
	ReleasedAt           *time.Time
}

// KotsKinds are all of the special "client-side" kinds that are packaged in
// an application. These should be pointers because they are all optional.
// But a few are still expected in the code later, so we make them not pointers,
// because other codepaths expect them to be present
type KotsKinds struct {
	KotsApplication kotsv1beta1.Application
	Application     *applicationv1beta1.Application
	HelmCharts      []*kotsv1beta1.HelmChart

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

// GetKustomizeBinaryPath will return the kustomize binary version to use for this application
// applying the default, if there is one, for the current version of kots
func (k KotsKinds) GetKustomizeBinaryPath() string {
	path, err := binaries.GetKustomizePathForVersion(k.KotsApplication.Spec.KustomizeVersion)
	if err != nil {
		logger.Infof("Failed to get kustomize path: %v", err)
		return "kustomize"
	}
	return path
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
					return "", nil
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

	return "", errors.Errorf("unknown gvk %s/%s, Kind=%s", g, v, k)
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

func (k *KotsKinds) GetImages() []string {
	if k == nil {
		return nil
	}

	allImages := []string{}

	allImages = append(allImages, k.KotsApplication.Spec.AdditionalImages...)

	collectors := make([]*troubleshootv1beta2.Collect, 0)
	if k.SupportBundle != nil {
		collectors = append(collectors, k.SupportBundle.Spec.Collectors...)
	}
	if k.Collector != nil {
		collectors = append(collectors, k.Collector.Spec.Collectors...)
	}
	if k.Preflight != nil {
		collectors = append(collectors, k.Preflight.Spec.Collectors...)
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
			if strings.Contains(image, "repl{{") || strings.Contains(image, "{{repl") {
				// Images that use templates like LocalImageName should be included in application's additionalImages list.
				// We want the original image names here only, not the templated ones.
				continue
			}
			allImages = append(allImages, image)
		}
	}

	return allImages
}

// create a new kots kinds, ensuring that the required objects exist as empty defaults
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
