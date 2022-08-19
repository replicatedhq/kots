package helm

import (
	"bytes"
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
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	kotsbase "github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/render"
	helmval "helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/helmpath"
	helmregistry "helm.sh/helm/v3/pkg/registry"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func RenderValuesFromConfig(helmApp *apptypes.HelmApp, kotsKinds *kotsutil.KotsKinds, chart []byte) (map[string]interface{}, error) {
	builder, err := render.NewBuilder(kotsKinds, registrytypes.RegistrySettings{}, helmApp.GetSlug(), helmApp.GetCurrentSequence(), helmApp.GetIsAirgap(), helmApp.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make tempalate builder")
	}

	renderedHelmManifest, err := builder.RenderTemplate("helm", string(chart))
	if err != nil {
		return nil, err
	}

	kotsHelmChart, err := kotsbase.ParseHelmChart([]byte(renderedHelmManifest))
	if err != nil {
		return nil, err
	}

	mergedValues := kotsHelmChart.Spec.Values
	for _, optionalValues := range kotsHelmChart.Spec.OptionalValues {
		include, err := strconv.ParseBool(optionalValues.When)
		if err != nil {
			return nil, err
		}
		if !include {
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
		return nil, err
	}

	return renderedValues, nil
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

func GetConfigValuesMap(configValues *kotsv1beta1.ConfigValues) (map[string]interface{}, error) {
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var configValuesBuffer bytes.Buffer
	if err := s.Encode(configValues, &configValuesBuffer); err != nil {
		return nil, errors.Wrap(err, "failed to encode config values")
	}

	configValuesMap := map[string]interface{}{
		"replicated": map[string]interface{}{
			"app": map[string][]byte{ // "byte" for base64 encoding
				"configValues": configValuesBuffer.Bytes(),
			},
		},
	}

	return configValuesMap, nil
}
