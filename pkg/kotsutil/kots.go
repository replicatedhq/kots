package kotsutil

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	troubleshootv1beta1 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta1"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	applicationv1beta1 "sigs.k8s.io/application/pkg/apis/app/v1beta1"
)

// KotsKinds are all of the special "client-side" kinds that are packaged in
// an application. These should be pointers because they are all optional.
// But a few are still expected in the code later, so we make them not pointers,
// because other codepaths expect them to be present
type KotsKinds struct {
	KotsApplication kotsv1beta1.Application
	Application     *applicationv1beta1.Application

	Collector *troubleshootv1beta1.Collector
	Preflight *troubleshootv1beta1.Preflight
	Analyzer  *troubleshootv1beta1.Analyzer

	Config       *kotsv1beta1.Config
	ConfigValues *kotsv1beta1.ConfigValues

	Installation kotsv1beta1.Installation
	License      *kotsv1beta1.License

	Backup *velerov1.Backup
}

// KustomizeVersion will return the kustomize version to use for this application
// applying the default, if there is one, for the current version of kots
func (k KotsKinds) KustomizeVersion() string {
	if k.KotsApplication.Spec.KustomizeVersion != "" {
		return k.KotsApplication.Spec.KustomizeVersion
	}

	return "3.5.4"
}

func (o KotsKinds) Marshal(g string, v string, k string) (string, error) {
	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
	velerov1.AddToScheme(scheme.Scheme)
	applicationv1beta1.AddToScheme(scheme.Scheme)

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
			}
		}
	}

	if g == "troubleshoot.replicated.com" {
		if v == "v1beta1" {
			switch k {
			case "Collector":
				if o.Collector == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.Collector, &b); err != nil {
					return "", errors.Wrap(err, "failed to encode collector")
				}
				return string(b.Bytes()), nil
			case "Analyzer":
				if o.Analyzer == nil {
					return "", nil
				}
				var b bytes.Buffer
				if err := s.Encode(o.Analyzer, &b); err != nil {
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

	return "", errors.Errorf("unknown gvk %s/%s, Kind=%s", g, v, k)
}

// create a new kots kinds, ensuring that the require objets exist as empty defaults
func emptyKotsKinds() KotsKinds {
	kotsKinds := KotsKinds{
		Installation: kotsv1beta1.Installation{
			Spec: kotsv1beta1.InstallationSpec{},
		},
		KotsApplication: kotsv1beta1.Application{
			Spec: kotsv1beta1.ApplicationSpec{},
		},
	}

	return kotsKinds
}

func LoadKotsKindsFromPath(fromDir string) (*KotsKinds, error) {
	kotsKinds := emptyKotsKinds()

	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
	velerov1.AddToScheme(scheme.Scheme)
	applicationv1beta1.AddToScheme(scheme.Scheme)

	decode := scheme.Codecs.UniversalDeserializer().Decode

	err := filepath.Walk(filepath.Join(fromDir, "upstream"),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			contents, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			decoded, gvk, err := decode(contents, nil, nil)
			if err != nil {
				return nil // not an error because the file might not be yaml
			}

			switch gvk.String() {
			case "kots.io/v1beta1, Kind=Config":
				kotsKinds.Config = decoded.(*kotsv1beta1.Config)
			case "kots.io/v1beta1, Kind=ConfigValues":
				kotsKinds.ConfigValues = decoded.(*kotsv1beta1.ConfigValues)
			case "kots.io/v1beta1, Kind=Application":
				kotsKinds.KotsApplication = *decoded.(*kotsv1beta1.Application)
			case "kots.io/v1beta1, Kind=License":
				kotsKinds.License = decoded.(*kotsv1beta1.License)
			case "kots.io/v1beta1, Kind=Installation":
				kotsKinds.Installation = *decoded.(*kotsv1beta1.Installation)
			case "troubleshoot.replicated.com/v1beta1, Kind=Collector":
				kotsKinds.Collector = decoded.(*troubleshootv1beta1.Collector)
			case "troubleshoot.replicated.com/v1beta1, Kind=Analyzer":
				kotsKinds.Analyzer = decoded.(*troubleshootv1beta1.Analyzer)
			case "troubleshoot.replicated.com/v1beta1, Kind=Preflight":
				kotsKinds.Preflight = decoded.(*troubleshootv1beta1.Preflight)
			case "velero.io/v1, Kind=Backup":
				kotsKinds.Backup = decoded.(*velerov1.Backup)
			}

			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk upstream dir")
	}

	return &kotsKinds, nil
}

func LoadInstallationFromPath(installationFilePath string) (*kotsv1beta1.Installation, error) {
	installationData, err := ioutil.ReadFile(installationFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read installation file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(installationData), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode installation data")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Installation" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.Installation), nil
}

func LoadLicenseFromPath(licenseFilePath string) (*kotsv1beta1.License, error) {
	licenseData, err := ioutil.ReadFile(licenseFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read license file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode([]byte(licenseData), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode license data")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "License" {
		return nil, errors.Errorf("unexpected GVK: %s", gvk.String())
	}

	return obj.(*kotsv1beta1.License), nil
}

func LoadConfigValuesFromFile(configValuesFilePath string) (*kotsv1beta1.ConfigValues, error) {
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

func LoadPreflightFromContents(content []byte) (*troubleshootv1beta1.Preflight, error) {
	troubleshootscheme.AddToScheme(scheme.Scheme)

	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "not a preflight")
	}

	if gvk.String() != "troubleshoot.replicated.com/v1beta1, Kind=Preflight" {
		return nil, errors.New("not a preflight")
	}

	return obj.(*troubleshootv1beta1.Preflight), nil
}
