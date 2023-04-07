package postrenderer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/chartutil"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

// TODO: add license, registry, app and other info necessary to render
type PostRenderer struct {
	kustomizeBinPath     string
	rootKustomizationDir string
	downstream           string
	releaseName          string
	namespace            string
	appSlug              string
	registryHost         string
	registryUsername     string
	registryPassword     string
	license              *kotsv1beta1.License
	kotsApplication      *kotsv1beta1.Application
}

type PostRendererOptions struct {
	KustomizeBinPath     string
	RootKustomizationDir string
	Downstream           string
	ReleaseName          string
	Namespace            string
	AppSlug              string
	RegistryHost         string
	RegistryUsername     string
	RegistryPassword     string
	License              *kotsv1beta1.License
	KotsApplication      *kotsv1beta1.Application
}

func NewPostRenderer(opts *PostRendererOptions) *PostRenderer {
	return &PostRenderer{
		kustomizeBinPath:     opts.KustomizeBinPath,
		rootKustomizationDir: opts.RootKustomizationDir,
		downstream:           opts.Downstream,
		releaseName:          opts.ReleaseName,
		namespace:            opts.Namespace,
		appSlug:              opts.AppSlug,
		registryHost:         opts.RegistryHost,
		registryUsername:     opts.RegistryUsername,
		registryPassword:     opts.RegistryPassword,
		license:              opts.License,
		kotsApplication:      opts.KotsApplication,
	}
}

func (r *PostRenderer) Run(renderedManifests *bytes.Buffer) (modifiedManifests *bytes.Buffer, err error) {
	basePath := path.Join(r.rootKustomizationDir, "base")
	if err := os.MkdirAll(basePath, 0744); err != nil {
		return nil, errors.Wrap(err, "failed to create base directory")
	}

	allYamlPath := path.Join(basePath, "all.yaml")
	if err := os.WriteFile(allYamlPath, renderedManifests.Bytes(), 0644); err != nil {
		return nil, errors.Wrap(err, "failed to write all.yaml")
	}

	baseKustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		Resources: []string{"all.yaml"},
	}

	baseKustomizationBytes, err := yaml.Marshal(baseKustomization)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal base kustomization.yaml")
	}

	baseKustomizationPath := path.Join(basePath, "kustomization.yaml")
	if err := os.WriteFile(baseKustomizationPath, baseKustomizationBytes, 0644); err != nil {
		return nil, errors.Wrap(err, "failed to write base kustomization.yaml")
	}

	midstreamPath := path.Join(r.rootKustomizationDir, "overlays", "midstream")
	if err := os.MkdirAll(midstreamPath, 0744); err != nil {
		return nil, errors.Wrap(err, "failed to create midstream directory")
	}

	baseRelPath, err := filepath.Rel(midstreamPath, basePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get relative path")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes clientset")
	}

	midstreamKustomization, err := CreateMidstreamKustomization(clientset, &CreateMidstreamKustomizationOptions{
		MidstreamPath:    midstreamPath,
		BaseRelPath:      baseRelPath,
		ReleaseName:      r.releaseName,
		Namespace:        r.namespace,
		AppSlug:          r.appSlug,
		RegistryHost:     r.registryHost,
		RegistryUsername: r.registryUsername,
		RegistryPassword: r.registryPassword,
		License:          r.license,
		KotsApplication:  r.kotsApplication,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create midstream kustomization")
	}

	// TODO: actually make the real midstream kustomization
	// 1. image pull secrets
	// 2. image rewrite
	// 3. backup labels
	// 4. other miscelaneous things we do with kustomize...

	midstreamKustomizationBytes, err := yaml.Marshal(midstreamKustomization)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal midstream kustomization.yaml")
	}

	midstreamKustomizationPath := path.Join(midstreamPath, "kustomization.yaml")
	if err := os.WriteFile(midstreamKustomizationPath, midstreamKustomizationBytes, 0644); err != nil {
		return nil, errors.Wrap(err, "failed to write midstream kustomization.yaml")
	}

	downstreamTarget := path.Join(r.rootKustomizationDir, "overlays", "downstreams", r.downstream)

	allContent, err := exec.Command(r.kustomizeBinPath, "build", downstreamTarget).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
		}
		return nil, errors.Wrap(err, "failed to run kustomize build")
	}

	return bytes.NewBuffer(allContent), nil
}

type CreatePullSecretsOptions struct {
	Namespace        string
	AppSlug          string
	ReleaseName      string
	RegistryHost     string
	RegistryUsername string
	RegistryPassword string
	License          *kotsv1beta1.License
	KotsApplication  *kotsv1beta1.Application
}

func CreatePullSecrets(clientset kubernetes.Interface, opts *CreatePullSecretsOptions) (*registry.ImagePullSecrets, error) {
	var pullSecretRegistries []string
	var pullSecretUsername string
	var pullSecretPassword string

	if opts.RegistryHost != "" {
		// Use target registry credentials to create image pull secrets for all objects that have images.
		pullSecretRegistries = []string{opts.RegistryHost}
		pullSecretUsername = opts.RegistryUsername
		pullSecretPassword = opts.RegistryPassword
		if pullSecretUsername == "" {
			newPullSecretUsername, newPullSecretPassword, err := registry.LoadAuthForRegistry(opts.RegistryHost)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load registry auth for %q", opts.RegistryHost)
			}
			pullSecretUsername, pullSecretPassword = newPullSecretUsername, newPullSecretPassword
		}
	} else if opts.License != nil {
		// Use license to create image pull secrets for all objects that have private images.
		pullSecretRegistries = registry.GetRegistryProxyInfo(opts.License, opts.KotsApplication).ToSlice()
		pullSecretUsername = opts.License.Spec.LicenseID
		pullSecretPassword = opts.License.Spec.LicenseID
	}

	namePrefix := fmt.Sprintf("%s-%s", opts.AppSlug, opts.ReleaseName)

	pullSecrets, err := registry.PullSecretForRegistries(pullSecretRegistries, pullSecretUsername, pullSecretPassword, opts.Namespace, namePrefix)
	if err != nil {
		return nil, errors.Wrap(err, "create pull secret")
	}

	// do not fail on being unable to get dockerhub credentials, since they're just used to increase the rate limit
	pullSecrets.DockerHubSecret, _ = registry.GetDockerHubPullSecret(clientset, util.PodNamespace, opts.Namespace, namePrefix)

	return &pullSecrets, nil
}

type CreateMidstreamKustomizationOptions struct {
	MidstreamPath    string
	BaseRelPath      string
	ReleaseName      string
	Namespace        string
	AppSlug          string
	RegistryHost     string
	RegistryUsername string
	RegistryPassword string
	License          *kotsv1beta1.License
	KotsApplication  *kotsv1beta1.Application
}

func CreateMidstreamKustomization(clientset kubernetes.Interface, opts *CreateMidstreamKustomizationOptions) (*kustomizetypes.Kustomization, error) {
	midstreamKustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		Resources: []string{
			opts.BaseRelPath,
		},
	}

	// TODO: Find out why this was necessary
	// existingKustomization, err := m.getExistingKustomization(options)
	// if err != nil {
	// 	return errors.Wrap(err, "get existing kustomization")
	// }

	pullSecrets, err := CreatePullSecrets(clientset, &CreatePullSecretsOptions{
		Namespace:        opts.Namespace, // TODO: this should be the helmChart namespace
		AppSlug:          opts.AppSlug,
		ReleaseName:      opts.ReleaseName,
		RegistryHost:     opts.RegistryHost,
		RegistryUsername: opts.RegistryUsername,
		RegistryPassword: opts.RegistryPassword,
		License:          opts.License,
		KotsApplication:  opts.KotsApplication,
	})

	// QUESTION: can admin console pull secret be nil since its created at the top level already and not used by the helm chart?
	secretFilename, err := midstream.WritePullSecret(opts.MidstreamPath, pullSecrets.AppSecret, nil, pullSecrets.DockerHubSecret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write pull secret")
	}

	if secretFilename != "" {
		midstreamKustomization.Resources = append(midstreamKustomization.Resources, secretFilename)
	}

	// if err := m.writeObjectsWithPullSecret(options); err != nil {
	// 	return errors.Wrap(err, "failed to write patches")
	// }

	// identityBase, err := m.writeIdentityService(context.TODO(), options)
	// if err != nil {
	// 	return errors.Wrap(err, "failed to write identity service")
	// }

	// if identityBase != "" {
	// 	m.Kustomization.Resources = append(m.Kustomization.Resources, identityBase)
	// }

	// transformers
	drLabelTransformerFilename, err := midstream.WriteDisasterRecoveryLabelTransformer(opts.AppSlug, opts.MidstreamPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write disaster recovery label transformer")
	}
	midstreamKustomization.Transformers = append(midstreamKustomization.Transformers, drLabelTransformerFilename)

	// annotations
	if midstreamKustomization.CommonAnnotations == nil {
		midstreamKustomization.CommonAnnotations = make(map[string]string)
	}
	midstreamKustomization.CommonAnnotations["kots.io/app-slug"] = opts.AppSlug

	// TODO: same as above
	// if existingKustomization != nil {
	// 	// Note that this function does nothing on the initial install
	// 	// if the user is not presented with the config screen.
	// 	m.mergeKustomization(options, *existingKustomization)
	// }

	midstreamKustomizationPath := path.Join(opts.MidstreamPath, "kustomization.yaml")
	if err := k8sutil.WriteKustomizationToFile(midstreamKustomization, midstreamKustomizationPath); err != nil {
		return nil, errors.Wrap(err, "failed to write kustomization.yaml")
	}

	return &midstreamKustomization, nil
}

func WriteHelmPostRendererDownstreams(rootDir string, downstreams []string, helmCharts []*kotsv1beta1.HelmChart, log *logger.CLILogger) error {
	for _, downstream := range downstreams {
		log.Debug("Writing helm post-renderer downstream %s", downstream)
		for _, helmChart := range helmCharts {
			// TODO: filter helm charts before this
			if !helmChart.Spec.UsePostRenderer {
				continue
			}

			downstreamDir := path.Join(rootDir, "helm", "charts", helmChart.GetDirName(), "overlays", "downstreams", downstream)
			if err := os.MkdirAll(downstreamDir, 0744); err != nil {
				return errors.Wrap(err, "failed to create downstream dir")
			}

			kustomizationPath := path.Join(downstreamDir, "kustomization.yaml")
			if _, err := os.Stat(kustomizationPath); err == nil {
				// kustomization.yaml already exists, skip
				continue
			}

			kustomization := kustomizetypes.Kustomization{
				TypeMeta: kustomizetypes.TypeMeta{
					APIVersion: "kustomize.config.k8s.io/v1beta1",
					Kind:       "Kustomization",
				},
				Resources: []string{
					"../../midstream",
				},
			}

			if err := k8sutil.WriteKustomizationToFile(kustomization, path.Join(downstreamDir, "kustomization.yaml")); err != nil {
				return errors.Wrap(err, "failed to write kustomization")
			}
		}
	}

	return nil
}

func WriteHelmPostRendererCharts(u *upstreamtypes.Upstream, renderOptions *base.RenderOptions, rootDir string, helmCharts []*kotsv1beta1.HelmChart) error {
	for _, helmChart := range helmCharts {
		// TODO: filter helm charts before this
		if !helmChart.Spec.UsePostRenderer {
			continue
		}

		chartDir := path.Join(rootDir, "helm", "charts", helmChart.GetDirName())
		if err := os.MkdirAll(chartDir, 0744); err != nil {
			return errors.Wrap(err, "failed to create chart dir")
		}

		_, archive, err := base.FindHelmChartArchiveInRelease(u.Files, helmChart)
		if err != nil {
			return errors.Wrap(err, "failed to find helm chart archive in release")
		}

		archivePath := path.Join(chartDir, fmt.Sprintf("%s-%s.tgz", helmChart.Spec.Chart.Name, helmChart.Spec.Chart.ChartVersion))
		if err := ioutil.WriteFile(archivePath, archive, 0644); err != nil {
			return errors.Wrap(err, "failed to write helm chart archive")
		}

		mergedValues := helmChart.Spec.Values
		if mergedValues == nil {
			mergedValues = map[string]kotsv1beta1.MappedChartValue{}
		}
		for _, optionalValues := range helmChart.Spec.OptionalValues {
			parsedBool, err := strconv.ParseBool(optionalValues.When)
			if err != nil {
				return errors.Wrap(err, "failed to parse when conditional on optional values")
			}
			if !parsedBool {
				continue
			}
			if optionalValues.RecursiveMerge {
				mergedValues = kotsv1beta1.MergeHelmChartValues(mergedValues, optionalValues.Values)
			} else {
				for k, v := range optionalValues.Values {
					mergedValues[k] = v
				}
			}
		}

		helmValues, err := helmChart.Spec.GetHelmValues(mergedValues)
		if err != nil {
			return errors.Wrap(err, "failed to render local values for chart")
		}

		valuesContent, err := yaml.Marshal(helmValues)
		if err != nil {
			return errors.Wrap(err, "failed to marshal values")
		}

		builder, _, err := base.NewConfigContextTemplateBuilder(u, renderOptions)
		if err != nil {
			return errors.Wrap(err, "failed to create config context template builder")
		}

		renderedValuesContent, err := builder.RenderTemplate(fmt.Sprintf("%s-values", helmChart.GetDirName()), string(valuesContent))
		if err != nil {
			return errors.Wrap(err, "failed to render values")
		}

		valuesPath := path.Join(chartDir, "values.yaml")
		if err := ioutil.WriteFile(valuesPath, []byte(renderedValuesContent), 0644); err != nil {
			return errors.Wrap(err, "failed to write values file")
		}
	}

	return nil
}

type WriteOptions struct {
	RootDir          string
	Downstreams      []string
	KustomizeBinPath string
	Log              *logger.CLILogger
	// Namespace        string
	AppSlug          string
	RegistryHost     string
	RegistryUsername string
	RegistryPassword string
	KotsKinds        *kotsutil.KotsKinds
}

func WriteRenderedApp(opts WriteOptions) error {
	for _, downstream := range opts.Downstreams {
		// TODO: filter the helm charts before this
		for _, helmChart := range opts.KotsKinds.HelmCharts {
			if !helmChart.Spec.UsePostRenderer {
				continue
			}

			// do a dry-run using the post-renderer to create the kustomize base
			cfg := &action.Configuration{
				Log: opts.Log.Debug,
			}
			client := action.NewInstall(cfg)
			client.DryRun = true
			client.ReleaseName = helmChart.GetReleaseName()
			client.Replace = true
			client.ClientOnly = true
			client.IncludeCRDs = true

			chartDir := path.Join(opts.RootDir, "helm", "charts", helmChart.GetDirName())

			client.PostRenderer = NewPostRenderer(&PostRendererOptions{
				KustomizeBinPath:     opts.KustomizeBinPath,
				RootKustomizationDir: chartDir,
				Downstream:           downstream,
				ReleaseName:          helmChart.GetReleaseName(),
				Namespace:            helmChart.Spec.Namespace,
				AppSlug:              opts.AppSlug,
				RegistryHost:         opts.RegistryHost,
				RegistryUsername:     opts.RegistryUsername,
				RegistryPassword:     opts.RegistryPassword,
				License:              opts.KotsKinds.License,
				KotsApplication:      &opts.KotsKinds.KotsApplication,
			})

			client.Namespace = helmChart.Spec.Namespace
			if client.Namespace == "" {
				client.Namespace = util.PodNamespace
			}

			chartPath := path.Join(chartDir, fmt.Sprintf("%s-%s.tgz", helmChart.Spec.Chart.Name, helmChart.Spec.Chart.ChartVersion))
			valuesPath := path.Join(chartDir, "values.yaml")

			chartRequested, err := loader.Load(chartPath)
			if err != nil {
				return errors.Wrap(err, "failed to load chart")
			}

			if req := chartRequested.Metadata.Dependencies; req != nil {
				if err := action.CheckDependencies(chartRequested, req); err != nil {
					return errors.Wrap(err, "failed dependency check")
				}
			}

			values, err := chartutil.ReadValuesFile(valuesPath)
			if err != nil {
				return errors.Wrap(err, "failed to read values file")
			}

			rel, err := client.Run(chartRequested, values)
			if err != nil {
				return errors.Wrap(err, "failed to run helm install")
			}

			var manifests bytes.Buffer
			fmt.Fprintln(&manifests, strings.TrimSpace(rel.Manifest))
			for _, m := range rel.Hooks {
				fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", m.Path, m.Manifest)
			}

			renderedPath := path.Join(opts.RootDir, "rendered", downstream, "charts", helmChart.GetDirName())
			if err := os.MkdirAll(renderedPath, 0744); err != nil {
				return errors.Wrap(err, "failed to create rendered path")
			}

			// TODO: This belongs in the post-renderer
			// kustomization := kustomizetypes.Kustomization{
			// 	TypeMeta: kustomizetypes.TypeMeta{
			// 		APIVersion: "kustomize.config.k8s.io/v1beta1",
			// 		Kind:       "Kustomization",
			// 	},
			// 	Resources: []string{"all.yaml"},
			// }
			// b, err := yaml.Marshal(kustomization)
			// if err != nil {
			// 	return errors.Wrap(err, "failed to marshal kustomization")
			// }

			// err = os.WriteFile(filepath.Join(renderedPath, "kustomization.yaml"), b, 0644)
			// if err != nil {
			// 	return errors.Wrap(err, "failed to write kustomization.yaml")
			// }

			// write all.yaml
			err = os.WriteFile(filepath.Join(renderedPath, "all.yaml"), manifests.Bytes(), 0644)
			if err != nil {
				return errors.Wrap(err, "failed to write all.yaml")
			}

			// TODO: at this point we have a base/all.yaml that can be used to pull and re-write images from online
		}
	}

	return nil
}

// GetPostRendererChartsArchive returns an archive of all charts to be post-rendered
func GetPostRendererChartsArchive(deployedVersionArchive string) ([]byte, error) {
	chartsDir := filepath.Join(deployedVersionArchive, "helm", "charts")

	// if the charts dir doesn't exist, return an empty archive
	if _, err := os.Stat(chartsDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to stat charts dir")
	}

	archive, err := util.TGZArchive(chartsDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create charts archive")
	}

	return archive, nil
}
