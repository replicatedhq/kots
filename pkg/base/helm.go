package base

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/strvals"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func RenderHelm(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (*Base, error) {
	chartPath, err := ioutil.TempDir("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chart dir")
	}
	defer os.RemoveAll(chartPath)

	for _, file := range u.Files {
		p := path.Join(chartPath, file.Path)
		d, _ := path.Split(p)
		if _, err := os.Stat(d); err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(d, 0744); err != nil {
					return nil, errors.Wrap(err, "failed to mkdir for chart resource")
				}
			} else {
				return nil, errors.Wrap(err, "failed to check if dir exists")
			}
		}

		if err := ioutil.WriteFile(p, file.Content, 0644); err != nil {
			return nil, errors.Wrap(err, "failed to write chart file")
		}
	}

	vals := map[string]interface{}{}
	for _, value := range renderOptions.HelmOptions {
		if err := strvals.ParseInto(value, vals); err != nil {
			return nil, errors.Wrapf(err, "failed to parse helm value %q", value)
		}
	}

	var rendered map[string]string
	switch strings.ToLower(renderOptions.HelmVersion) {
	case "v3":
		rendered, err = renderHelmV3(u.Name, chartPath, vals, renderOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render with helm v3")
		}
	case "v2", "":
		rendered, err = renderHelmV2(u.Name, chartPath, vals, renderOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render with helm v2")
		}
	default:
		return nil, errors.Errorf("unknown helmVersion %s", renderOptions.HelmVersion)
	}

	baseFiles := []BaseFile{}
	for k, v := range rendered {
		var fileStrings []string
		if renderOptions.SplitMultiDocYAML {
			fileStrings = strings.Split(v, "\n---\n")
		} else {
			fileStrings = append(fileStrings, v)
		}

		if len(fileStrings) == 1 {
			content, err := transpileHelmHooksToKotsHooks([]byte(v))
			if err != nil {
				return nil, errors.Wrap(err, "failed to transpile helm hooks to kots hooks")
			}

			baseFiles = append(baseFiles, BaseFile{
				Path:    k,
				Content: content,
			})
			continue
		}

		for idx, fileString := range fileStrings {
			filename := strings.TrimSuffix(k, filepath.Ext(k))
			filename = fmt.Sprintf("%s-%d%s", filename, idx+1, filepath.Ext(k))

			content, err := transpileHelmHooksToKotsHooks([]byte(fileString))
			if err != nil {
				return nil, errors.Wrap(err, "failed to transpile helm hooks to kots hooks")
			}

			baseFiles = append(baseFiles, BaseFile{
				Path:    filename,
				Content: content,
			})
		}
	}

	baseFiles = removeCommonPrefix(baseFiles)

	return &Base{
		Files: baseFiles,
	}, nil
}

func removeCommonPrefix(baseFiles []BaseFile) []BaseFile {
	// remove any common prefix from all files
	if len(baseFiles) == 0 {
		return baseFiles
	}

	firstFileDir, _ := path.Split(baseFiles[0].Path)
	commonPrefix := strings.Split(firstFileDir, string(os.PathSeparator))

	for _, file := range baseFiles {
		d, _ := path.Split(file.Path)
		dirs := strings.Split(d, string(os.PathSeparator))

		commonPrefix = util.CommonSlicePrefix(commonPrefix, dirs)
	}

	cleanedBaseFiles := []BaseFile{}
	for _, file := range baseFiles {
		d, f := path.Split(file.Path)
		d2 := strings.Split(d, string(os.PathSeparator))

		cleanedBaseFile := file
		d2 = d2[len(commonPrefix):]
		cleanedBaseFile.Path = path.Join(path.Join(d2...), f)

		cleanedBaseFiles = append(cleanedBaseFiles, cleanedBaseFile)
	}

	return cleanedBaseFiles
}

func transpileHelmHooksToKotsHooks(content []byte) ([]byte, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(content, nil, nil)
	if err != nil {
		return content, nil // this isn't an error, it's just not a job witih a hook, that's certain
	}

	annotations, _ := metadataAccessor.Annotations(obj)

	var annotationsUpdated bool

	if value, ok := annotations[release.HookAnnotation]; ok {
		annotations[HookAnnotation] = value
		annotationsUpdated = true
	}

	if value, ok := annotations[release.HookWeightAnnotation]; ok {
		annotations[HookWeightAnnotation] = value
		annotationsUpdated = true
	}

	if value, ok := annotations[release.HookDeleteAnnotation]; ok {
		annotations[HookDeleteAnnotation] = value
		annotationsUpdated = true
	}

	if !annotationsUpdated {
		return content, nil
	}

	if err := metadataAccessor.SetAnnotations(obj, annotations); err != nil {
		return content, errors.Wrap(err, "failed to set kots.io hook annotations")
	}

	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(obj, &b); err != nil {
		return content, errors.Wrap(err, "failed to encode job")
	}

	return b.Bytes(), nil
}
