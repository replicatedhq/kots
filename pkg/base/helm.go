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
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/releaseutil"
	"helm.sh/helm/v3/pkg/strvals"
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

	// config := &chart.Config{Raw: string(marshalledVals), Values: map[string]*chart.Value{}}

	// renderOpts := renderutil.Options{
	// 	ReleaseOptions: chartutil.ReleaseOptions{
	// 		Name:      u.Name,
	// 		IsInstall: true,
	// 		IsUpgrade: false,
	// 		Time:      timeconv.Now(),
	// 		Namespace: renderOptions.Namespace,
	// 	},
	// 	KubeVersion: "1.16.0",
	// }

	cfg := &action.Configuration{
		Log: renderOptions.Log.Debug,
	}
	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = u.Name
	client.Replace = true
	client.ClientOnly = true
	// client.IncludeCRDs = includeCrds
	client.Namespace = renderOptions.Namespace

	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load chart")
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			return nil, errors.Wrap(err, "failed dependency check")
		}
	}

	rel, err := client.Run(chartRequested, vals)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render chart")
	}

	var manifests bytes.Buffer
	fmt.Fprintln(&manifests, strings.TrimSpace(rel.Manifest))
	for _, m := range rel.Hooks {
		fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", m.Path, m.Manifest)
	}
	rendered := releaseutil.SplitManifests(manifests.String())

	// Silence the go logger because helm will complain about some of our template strings
	// golog.SetOutput(ioutil.Discard)
	// defer golog.SetOutput(os.Stdout)
	// rendered, err := renderutil.Render(chartRequested, config, renderOpts)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "failed to render chart")
	// }

	baseFiles := []BaseFile{}
	for k, v := range rendered {
		if !renderOptions.SplitMultiDocYAML {
			baseFile := BaseFile{
				Path:    k,
				Content: []byte(v),
			}
			if err := baseFile.transpileHelmHooksToKotsHooks(); err != nil {
				return nil, errors.Wrap(err, "failed to transpile helm hooks to kots hooks")
			}

			baseFiles = append(baseFiles, baseFile)
			continue
		}

		fileStrings := strings.Split(v, "\n---\n")
		if len(fileStrings) == 1 {
			baseFile := BaseFile{
				Path:    k,
				Content: []byte(v),
			}
			if err := baseFile.transpileHelmHooksToKotsHooks(); err != nil {
				return nil, errors.Wrap(err, "failed to transpile helm hooks to kots hooks")
			}

			baseFiles = append(baseFiles, baseFile)
			continue
		}

		for idx, fileString := range fileStrings {
			filename := strings.TrimSuffix(k, filepath.Ext(k))
			filename = fmt.Sprintf("%s-%d%s", filename, idx+1, filepath.Ext(k))

			baseFile := BaseFile{
				Path:    filename,
				Content: []byte(fileString),
			}
			if err := baseFile.transpileHelmHooksToKotsHooks(); err != nil {
				return nil, errors.Wrap(err, "failed to transpile helm hooks to kots hooks")
			}

			baseFiles = append(baseFiles, baseFile)
		}
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
