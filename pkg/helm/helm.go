package helm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	imagedocker "github.com/containers/image/v5/docker"
	dockerref "github.com/containers/image/v5/docker/reference"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsbase "github.com/replicatedhq/kots/pkg/base"
	kotsconfig "github.com/replicatedhq/kots/pkg/config"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	helmval "helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/helmpath"
	helmregistry "helm.sh/helm/v3/pkg/registry"
)

func RenderValuesFromConfig(app string, newConfigItems map[string]template.ItemValue, config *kotsv1beta1.Config, chart []byte) (map[string]interface{}, *kotsv1beta1.Config, error) {
	renderedConfig, err := kotsconfig.TemplateConfigObjects(config, newConfigItems, nil, nil, template.LocalRegistry{}, nil, &template.ApplicationInfo{Slug: app}, nil, util.PodNamespace, true)
	if err != nil || renderedConfig == nil || len(renderedConfig.Spec.Groups) == 0 {
		return nil, nil, err
	}

	opts := template.BuilderOptions{
		ConfigGroups:    renderedConfig.Spec.Groups,
		ApplicationInfo: &template.ApplicationInfo{Slug: app},
		ExistingValues:  newConfigItems,
		LocalRegistry:   template.LocalRegistry{},
		License:         nil,
		Application:     &kotsv1beta1.Application{},
		VersionInfo:     &template.VersionInfo{},
		IdentityConfig:  &kotsv1beta1.IdentityConfig{},
		Namespace:       util.PodNamespace,
		DecryptValues:   true,
	}
	builder, _, err := template.NewBuilder(opts)
	if err != nil {
		return nil, renderedConfig, err
	}

	renderedHelmManifest, err := builder.RenderTemplate("helm", string(chart))
	if err != nil {
		return nil, renderedConfig, err
	}

	kotsHelmChart, err := kotsbase.ParseHelmChart([]byte(renderedHelmManifest))
	if err != nil {
		return nil, renderedConfig, err
	}

	mergedValues := kotsHelmChart.Spec.Values
	for _, optionalValues := range kotsHelmChart.Spec.OptionalValues {
		parsedBool, err := strconv.ParseBool(optionalValues.When)
		if err != nil {
			return nil, renderedConfig, err
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

	renderedValues, err := kotsHelmChart.Spec.GetHelmValues(mergedValues)
	if err != nil {
		return nil, renderedConfig, err
	}

	return renderedValues, renderedConfig, nil
}

func GetMergedValues(releasedValues, renderedValues map[string]interface{}) (map[string]interface{}, error) {
	dir, err := ioutil.TempDir("", "helm-merged-values-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	releasedB, err := json.Marshal(releasedValues)
	if err != nil {
		return nil, err

	}
	renderedB, err := json.Marshal(renderedValues)
	if err != nil {
		return nil, err
	}
	releaseValsFilename := fmt.Sprintf("%s/releasevalues.yaml", dir)
	renderedValsFilename := fmt.Sprintf("%s/renderedvalues.yaml", dir)
	if err := ioutil.WriteFile(releaseValsFilename, releasedB, 0644); err != nil {
		return nil, err
	}

	if err := ioutil.WriteFile(renderedValsFilename, renderedB, 0644); err != nil {
		return nil, err
	}

	helmopts := &helmval.Options{ValueFiles: []string{releaseValsFilename, renderedValsFilename}}
	mergedHelmVals, err := helmopts.MergeValues(nil)
	if err != nil {
		return nil, err
	}

	return mergedHelmVals, nil
}

func CreateHelmRegistryCreds(username string, password string, url string) error {
	url = strings.TrimLeft(url, "oci://")
	ref, err := imagedocker.ParseReference(fmt.Sprintf("//%s", url))
	if err != nil {
		return errors.Wrapf(err, "failed to parse support bundle ref %q", url)
	}
	dockerRef := ref.DockerReference()

	registryHost := dockerref.Domain(dockerRef)

	dockercfgAuth := registry.DockercfgAuth{
		Auth: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
	}

	dockerCfgJSON := registry.DockerCfgJSON{
		Auths: map[string]registry.DockercfgAuth{
			registryHost: dockercfgAuth,
		},
	}
	data, err := json.MarshalIndent(dockerCfgJSON, "", "   ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal helm registry credentials")
	}

	filename := helmpath.ConfigPath(helmregistry.CredentialsFileBasename)

	err = os.MkdirAll(filepath.Dir(filename), 0700)
	if err != nil {
		return errors.Wrap(err, "failed to create directory for helm registry credentials")
	}

	err = ioutil.WriteFile(filename, data, 0600)
	if err != nil {
		return errors.Wrap(err, "failed to save helm registry credentials")
	}

	return nil
}
