package base

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/strvals"
	"k8s.io/helm/pkg/timeconv"
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
			return nil, errors.Wrap(err, "failed to parse helm value")
		}
	}
	marshalledVals, err := yaml.Marshal(vals)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal helm values")
	}

	config := &chart.Config{Raw: string(marshalledVals), Values: map[string]*chart.Value{}}

	c, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load chart")
	}

	renderOpts := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      u.Name,
			IsInstall: true,
			IsUpgrade: false,
			Time:      timeconv.Now(),
			Namespace: renderOptions.Namespace,
		},
		KubeVersion: "1.16.0",
	}

	rendered, err := renderutil.Render(c, config, renderOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render chart")
	}

	baseFiles := []BaseFile{}
	for k, v := range rendered {
		baseFile := BaseFile{
			Path:    k,
			Content: []byte(v),
		}

		baseFiles = append(baseFiles, baseFile)
	}

	// remove any common prefix from all files
	if len(baseFiles) > 0 {
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

		baseFiles = cleanedBaseFiles
	}

	return &Base{
		Files: baseFiles,
	}, nil
}
