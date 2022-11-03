package kotsutil

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	types "github.com/replicatedhq/kots/pkg/kotsutil/types"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	kurlscheme "github.com/replicatedhq/kurl/kurlkinds/client/kurlclientset/scheme"
	kurlv1beta1 "github.com/replicatedhq/kurl/kurlkinds/pkg/apis/cluster/v1beta1"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"github.com/replicatedhq/troubleshoot/pkg/docrewrite"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"gopkg.in/yaml.v2"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// LoadKotsKinds finds and renders (when applicable) all kots kinds from a path
func LoadKotsKinds(opts types.LoadKotsKindsOptions) (*types.KotsKinds, error) {
	kotsKinds := types.EmptyKotsKinds()

	if opts.FromDir == "" {
		return &kotsKinds, nil
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	err := filepath.Walk(opts.FromDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if ext := filepath.Ext(path); ext != ".yaml" && ext != ".yml" {
				return nil
			}

			contents, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			// kots kinds could be part of a multi-yaml doc
			allDocs := util.ConvertToSingleDocs(contents)

			// filter out non-kots kinds and create a map[gvk]doc for kots kinds
			kotsKindsMap := map[string][]byte{}
			for _, doc := range allDocs {
				gvk := types.OverlySimpleGVK{}
				if err := yaml.Unmarshal(doc, &gvk); err != nil {
					return errors.Wrap(err, "failed to unmarshal")
				}
				if IsKotsKind(gvk.APIVersion, gvk.Kind) {
					gvkString := fmt.Sprintf("%s, Kind=%s", gvk.APIVersion, gvk.Kind)
					kotsKindsMap[gvkString] = doc
				}
			}

			// we first need to detect the kots kinds that do not support templating and are needed for templating (config is a special case)
			for gvkString, doc := range kotsKindsMap {
				switch gvkString {
				case "kots.io/v1beta1, Kind=Config":
					decoded, _, err := decode(doc, nil, nil)
					if err != nil {
						return errors.Wrap(err, "failed to decode config")
					}
					kotsKinds.Config = decoded.(*kotsv1beta1.Config)
				case "kots.io/v1beta1, Kind=ConfigValues":
					decoded, _, err := decode(doc, nil, nil)
					if err != nil {
						return errors.Wrap(err, "failed to decode config values")
					}
					kotsKinds.ConfigValues = decoded.(*kotsv1beta1.ConfigValues)
				case "kots.io/v1beta1, Kind=Installation":
					decoded, _, err := decode(doc, nil, nil)
					if err != nil {
						return errors.Wrap(err, "failed to decode installation")
					}
					kotsKinds.Installation = *decoded.(*kotsv1beta1.Installation)
				case "kots.io/v1beta1, Kind=License":
					decoded, _, err := decode(doc, nil, nil)
					if err != nil {
						return errors.Wrap(err, "failed to decode license")
					}
					kotsKinds.License = decoded.(*kotsv1beta1.License)
				case "kots.io/v1beta1, Kind=IdentityConfig":
					decoded, _, err := decode(doc, nil, nil)
					if err != nil {
						return errors.Wrap(err, "failed to decode identity config")
					}
					kotsKinds.IdentityConfig = decoded.(*kotsv1beta1.IdentityConfig)
				}
			}

			builderOptions := types.BuilderOptions{
				Config:           kotsKinds.Config,
				ConfigValues:     kotsKinds.ConfigValues,
				Installation:     kotsKinds.Installation,
				License:          kotsKinds.License,
				IdentityConfig:   kotsKinds.IdentityConfig,
				RegistrySettings: opts.RegistrySettings,
				AppSlug:          opts.AppSlug,
				Sequence:         opts.Sequence,
				IsAirgap:         opts.IsAirgap,
				Namespace:        opts.Namespace,
			}
			builder, err := NewBuilder(builderOptions)
			if err != nil {
				return errors.Wrap(err, "failed to get template builder")
			}

			// the kots application kind can contain information that is needed for templating (e.g. proxyPublicImages),
			// but it also needs to be templated / rendered, so we first render it, then add it to the template context, then update the template builder
			if kotsAppDoc, ok := kotsKindsMap["kots.io/v1beta1, Kind=Application"]; ok {
				fixedUpContent, err := FixUpYAML(kotsAppDoc)
				if err != nil {
					return errors.Wrap(err, "failed to fix up yaml")
				}

				rendered, err := builder.RenderTemplate("kotskind", string(fixedUpContent))
				if err != nil {
					return errors.Wrap(err, "failed to render doc")
				}

				decoded, _, err := decode([]byte(rendered), nil, nil)
				if err != nil {
					return errors.Wrap(err, "failed to decode rendered kots application doc")
				}

				kotsKinds.KotsApplication = *decoded.(*kotsv1beta1.Application)

				// now that we have the kots application, we can update the template builder
				builderOptions.KotsApplication = kotsKinds.KotsApplication
				builder, err = NewBuilder(builderOptions)
				if err != nil {
					return errors.Wrap(err, "failed to update template builder")
				}
			}

			// now that we have a builder, render the doc before decoding it and parsing it as a kots kind
			for _, doc := range kotsKindsMap {
				fixedUpContent, err := FixUpYAML(doc)
				if err != nil {
					return errors.Wrap(err, "failed to fix up yaml")
				}

				rendered, err := builder.RenderTemplate("kotskind", string(fixedUpContent))
				if err != nil {
					return errors.Wrap(err, "failed to render doc")
				}

				decoded, gvk, err := decode([]byte(rendered), nil, nil)
				if err != nil {
					return errors.Wrap(err, "failed to decode rendered doc")
				}

				if strings.HasPrefix(gvk.String(), "troubleshoot.replicated.com/v1beta1,") {
					doc, err = docrewrite.ConvertToV1Beta2(doc)
					if err != nil {
						return errors.Wrap(err, "failed to convert to v1beta2")
					}
					decoded, gvk, err = decode(doc, nil, nil)
					if err != nil {
						return err
					}
				}

				switch gvk.String() {
				case "kots.io/v1beta1, Kind=Identity":
					kotsKinds.Identity = decoded.(*kotsv1beta1.Identity)
				case "kots.io/v1beta1, Kind=HelmChart":
					kotsKinds.HelmCharts = append(kotsKinds.HelmCharts, decoded.(*kotsv1beta1.HelmChart))
				case "kots.io/v1beta1, Kind=LintConfig":
					kotsKinds.LintConfig = decoded.(*kotsv1beta1.LintConfig)
				case "troubleshoot.sh/v1beta2, Kind=Collector":
					kotsKinds.Collector = decoded.(*troubleshootv1beta2.Collector)
				case "troubleshoot.sh/v1beta2, Kind=Analyzer":
					kotsKinds.Analyzer = decoded.(*troubleshootv1beta2.Analyzer)
				case "troubleshoot.sh/v1beta2, Kind=SupportBundle":
					kotsKinds.SupportBundle = decoded.(*troubleshootv1beta2.SupportBundle)
				case "troubleshoot.sh/v1beta2, Kind=Redactor":
					kotsKinds.Redactor = decoded.(*troubleshootv1beta2.Redactor)
				case "troubleshoot.sh/v1beta2, Kind=Preflight":
					kotsKinds.Preflight = decoded.(*troubleshootv1beta2.Preflight)
				case "troubleshoot.sh/v1beta2, Kind=HostPreflight":
					kotsKinds.HostPreflight = decoded.(*troubleshootv1beta2.HostPreflight)
				case "velero.io/v1, Kind=Backup":
					kotsKinds.Backup = decoded.(*velerov1.Backup)
				case "kurl.sh/v1beta1, Kind=Installer", "cluster.kurl.sh/v1beta1, Kind=Installer":
					kotsKinds.Installer = decoded.(*kurlv1beta1.Installer)
				case "app.k8s.io/v1beta1, Kind=Application":
					kotsKinds.Application = decoded.(*applicationv1beta1.Application)
				}
			}

			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk upstream dir")
	}

	return &kotsKinds, nil
}

func IsKotsKind(apiVersion string, kind string) bool {
	if apiVersion == "velero.io/v1" && kind == "Backup" {
		return true
	}
	if apiVersion == "kots.io/v1beta1" {
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
	// In addition to kotskinds, we exclude the application crd for now
	if apiVersion == "app.k8s.io/v1beta1" {
		return true
	}
	return false
}

// NewBuilder is a convenience function to create a new template builder from kots kinds and app metadata
func NewBuilder(opts types.BuilderOptions) (*template.Builder, error) {
	localRegistry := template.LocalRegistry{
		Host:      opts.RegistrySettings.Hostname,
		Namespace: opts.RegistrySettings.Namespace,
		Username:  opts.RegistrySettings.Username,
		Password:  opts.RegistrySettings.Password,
		ReadOnly:  opts.RegistrySettings.IsReadOnly,
	}

	templateContextValues := make(map[string]template.ItemValue)
	if opts.ConfigValues != nil {
		for k, v := range opts.ConfigValues.Spec.Values {
			templateContextValues[k] = template.ItemValue{
				Value:   v.Value,
				Default: v.Default,
			}
		}
	}

	err := crypto.InitFromString(opts.Installation.Spec.EncryptionKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load encryption cipher")
	}

	configGroups := []kotsv1beta1.ConfigGroup{}
	if opts.Config != nil && opts.Config.Spec.Groups != nil {
		configGroups = opts.Config.Spec.Groups
	}

	appInfo := template.ApplicationInfo{
		Slug: opts.AppSlug,
	}

	versionInfo := template.VersionInfoFromInstallation(opts.Sequence, opts.IsAirgap, opts.Installation.Spec)

	builderOptions := template.BuilderOptions{
		ConfigGroups:    configGroups,
		ExistingValues:  templateContextValues,
		LocalRegistry:   localRegistry,
		License:         opts.License,
		Application:     &opts.KotsApplication,
		ApplicationInfo: &appInfo,
		VersionInfo:     &versionInfo,
		IdentityConfig:  opts.IdentityConfig,
		Namespace:       opts.Namespace,
		DecryptValues:   true,
	}
	builder, _, err := template.NewBuilder(builderOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create template builder")
	}

	return &builder, nil
}

func LoadHelmChartFromContents(data []byte) (*kotsv1beta1.HelmChart, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, gvk, err := decode(data, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode helm chart")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "HelmChart" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return decoded.(*kotsv1beta1.HelmChart), nil
}

func LoadInstallationFromPath(installationFilePath string) (*kotsv1beta1.Installation, error) {
	installationData, err := ioutil.ReadFile(installationFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read installation file")
	}

	return LoadInstallationFromContents(installationData)
}

func LoadInstallationFromContents(installationData []byte) (*kotsv1beta1.Installation, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(installationData), nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode installation data of length %d", len(installationData))
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Installation" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.Installation), nil
}

func LoadK8sAppFromContents(data []byte) (*applicationv1beta1.Application, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(data), nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode k8s app data of length %d", len(data))
	}

	if gvk.Group != "app.k8s.io" || gvk.Version != "v1beta1" || gvk.Kind != "Application" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*applicationv1beta1.Application), nil
}

func LoadKotsAppFromContents(data []byte) (*kotsv1beta1.Application, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(data), nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode kots app data of length %d", len(data))
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Application" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.Application), nil
}

func LoadLicenseFromPath(licenseFilePath string) (*kotsv1beta1.License, error) {
	licenseData, err := ioutil.ReadFile(licenseFilePath)
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

func LoadConfigValuesFromPath(configValuesFilePath string) (*kotsv1beta1.ConfigValues, error) {
	configValuesData, err := ioutil.ReadFile(configValuesFilePath)
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

	isKurl, err := kurl.IsKurl()
	if err != nil {
		return autoConfig, errors.Wrap(err, "failed to check if cluster is kurl")
	}

	autoConfig.EnableImageDeletion = isKurl

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

func LoadIdentityConfigFromPath(identityConfigFilePath string) (*kotsv1beta1.IdentityConfig, error) {
	identityConfigData, err := ioutil.ReadFile(identityConfigFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read identityConfig file")
	}

	return LoadIdentityConfigFromContents(identityConfigData)
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
	files, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read airgap directory content")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		contents, err := ioutil.ReadFile(filepath.Join(root, file.Name()))
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

			contents, err := ioutil.ReadFile(path)
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
