package helm

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/Masterminds/semver"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/k8sdeps/validator"
	"sigs.k8s.io/kustomize/v3/pkg/commands/build"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
	"sigs.k8s.io/kustomize/v3/pkg/plugins"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
)

type MakeHelmChartOptions struct {
	KotsAppDir       string
	KustomizationDir string
	RenderDir        string
}

func MakeHelmChart(options MakeHelmChartOptions) error {
	_, chartName := path.Split(options.KotsAppDir)

	kustomizeBuildOutput, err := ioutil.TempDir("", "kots")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(kustomizeBuildOutput)

	resourceFactory := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())
	buildOptions := build.NewOptions(options.KustomizationDir, kustomizeBuildOutput)

	pluginConfig := plugins.DefaultPluginConfig()
	pluginLoader := plugins.NewLoader(pluginConfig, resourceFactory)

	var b bytes.Buffer
	if err := buildOptions.RunBuild(&b, validator.NewKustValidator(), fs.MakeRealFS(), resourceFactory, transformer.NewFactoryImpl(), pluginLoader); err != nil {
		return errors.Wrap(err, "failed to run kustomize build")
	}

	installationData, err := ioutil.ReadFile(path.Join(options.KotsAppDir, "upstream", "userdata", "installation.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to get installation data")
	}
	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(installationData), nil, nil)
	if err != nil {
		return errors.Wrap(err, "failed to decode installation data")
	}
	installation := obj.(*kotsv1beta1.Installation)

	appVersion := "1.0.0"
	parsedAppVersion, err := semver.NewVersion(installation.Spec.VersionLabel)
	if err == nil {
		appVersion = parsedAppVersion.Original()
	}

	applicationName, applicastionIcon, err := findApplicationNameAndIcon(path.Join(options.KotsAppDir, "upstream"))
	if err != nil {
		return errors.Wrap(err, "failed to find application name and icon")
	}

	chart := &chart.Metadata{
		Name:        chartName,
		Description: applicationName,
		Version:     fmt.Sprintf("1.0.%s", installation.Spec.UpdateCursor),
		AppVersion:  appVersion,
		ApiVersion:  chartutil.ApiVersionV1,
		Icon:        applicastionIcon,
	}

	_, err = chartutil.Create(chart, options.RenderDir)
	if err != nil {
		return errors.Wrap(err, "failed to render chart")
	}

	values, err := kotsConfigToHelmValues(path.Join(options.KotsAppDir, "upstream"))
	if err != nil {
		return errors.Wrap(err, "failed to turn kots config into helm values")
	}

	if err := ioutil.WriteFile(path.Join(options.RenderDir, chartName, "values.yaml"), values, 0644); err != nil {
		return errors.Wrap(err, "failed to write values")
	}

	return nil
}

func findApplicationNameAndIcon(upstreamDir string) (string, string, error) {
	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode

	applicationName := ""
	applicationIcon := ""

	err := filepath.Walk(upstreamDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			fileData, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			obj, gvk, err := decode([]byte(fileData), nil, nil)
			if err != nil {
				return nil // skip, might not be yaml
			}

			if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Application" {
				application := obj.(*kotsv1beta1.Application)

				applicationName = application.Spec.Title
				applicationIcon = application.Spec.Icon
			}

			return nil
		})

	if err != nil {
		return "", "", errors.Wrap(err, "failed to walk upstream dir")
	}

	return applicationName, applicationIcon, nil
}

func kotsConfigToHelmValues(upstreamDir string) ([]byte, error) {
	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode

	var values []byte

	err := filepath.Walk(upstreamDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			fileData, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			obj, gvk, err := decode([]byte(fileData), nil, nil)
			if err != nil {
				return nil // skip, might not be yaml
			}

			if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Config" {
				config := obj.(*kotsv1beta1.Config)

				simple := map[string]interface{}{}
				for _, group := range config.Spec.Groups {
					items := map[string]string{}

					for _, item := range group.Items {
						items[item.Name] = ""
					}

					simple[group.Name] = items
				}

				b, err := yaml.Marshal(simple)
				if err != nil {
					return err
				}

				values = b
			}

			return nil
		})

	if err != nil {
		return nil, errors.Wrap(err, "failed to walk upstream dir")
	}

	return values, nil
}
