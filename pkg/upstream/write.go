package upstream

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type WriteOptions struct {
	RootDir      string
	CreateAppDir bool
}

func (u *Upstream) WriteUpstream(options WriteOptions) error {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	renderDir = path.Join(renderDir, "upstream")

	var previousValuesContent []byte
	var previousAdminConsoleManifests map[string][]byte

	_, err := os.Stat(renderDir)
	if err == nil {
		// if there's already a values yaml, we need to save
		_, err := os.Stat(path.Join(renderDir, "userdata/values.yaml"))
		if err == nil {
			c, err := ioutil.ReadFile(path.Join(renderDir, "userdata/values.yaml"))
			if err != nil {
				return errors.Wrap(err, "failed to read existing values")
			}

			previousValuesContent = c
		}

		// we also save the admin console upstream from before
		adminConsoleManifests, err := ioutil.ReadDir(path.Join(renderDir, "admin-console"))
		if err == nil {
			previousAdminConsoleManifests = map[string][]byte{}
			for _, adminConsoleManifest := range adminConsoleManifests {
				if adminConsoleManifest.IsDir() {
					continue
				}

				content, err := ioutil.ReadFile(path.Join(renderDir, "admin-console", adminConsoleManifest.Name()))
				if err != nil {
					return errors.Wrap(err, "failed to read previous admin console file")
				}

				if adminConsoleManifest.Name() == "secret-jwt.yaml" {
					fmt.Printf("%s\n", content)
				}
				previousAdminConsoleManifests[adminConsoleManifest.Name()] = content
			}
		}

		if err := os.RemoveAll(renderDir); err != nil {
			return errors.Wrap(err, "failed to remove previous content in upstream")
		}
	}

	for fileIdx, file := range u.Files {
		fileRenderPath := path.Join(renderDir, file.Path)
		d, f := path.Split(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0744); err != nil {
				return errors.Wrap(err, "failed to mkdir")
			}
		}

		// if it's an admin console file, see if we have it from before
		pathSplit := strings.Split(file.Path, string(os.PathSeparator))
		if pathSplit[0] == "admin-console" {
			prevContent, ok := previousAdminConsoleManifests[f]
			if ok {
				if err := ioutil.WriteFile(fileRenderPath, prevContent, 0644); err != nil {
					return errors.Wrap(err, "failed to write upstream file")
				}

				u.Files[fileIdx].Content = prevContent

				goto Wrote
			}
		}

		if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
			return errors.Wrap(err, "failed to write upstream file")
		}
	}

	if previousValuesContent != nil {
		for i, f := range u.Files {
			if f.Path == "userdata/values.yaml" {
				mergedValues, err := mergeValues(previousValuesContent, f.Content)
				if err != nil {
					return errors.Wrap(err, "failed to merge values")
				}

				err = ioutil.WriteFile(path.Join(renderDir, "userdata/values.yaml"), mergedValues, 0644)
				if err != nil {
					return errors.Wrap(err, "failed to replace values with previous values")
				}

				updatedValues := UpstreamFile{
					Path:    f.Path,
					Content: mergedValues,
				}

				u.Files[i] = updatedValues
			}
		}
	}

	return nil
}

func (u *Upstream) GetBaseDir(options WriteOptions) string {
	renderDir := options.RootDir
	if options.CreateAppDir {
		renderDir = path.Join(renderDir, u.Name)
	}

	return path.Join(renderDir, "base")
}

func mergeValues(previousValues []byte, applicationDeliveredValues []byte) ([]byte, error) {
	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode

	prevObj, _, err := decode(previousValues, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode previoius values")
	}
	prevValues := prevObj.(*kotsv1beta1.ConfigValues)

	applicationValuesObj, _, err := decode(applicationDeliveredValues, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode application delivered values")
	}
	applicationValues := applicationValuesObj.(*kotsv1beta1.ConfigValues)

	for name, value := range applicationValues.Spec.Values {
		_, ok := prevValues.Spec.Values[name]
		if !ok {
			prevValues.Spec.Values[name] = value
		}
	}

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(prevValues, &b); err != nil {
		return nil, errors.Wrap(err, "failed to encode merged values")
	}

	return b.Bytes(), nil
}
