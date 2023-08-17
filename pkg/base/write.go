package base

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/util"
	"gopkg.in/yaml.v2"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type WriteOptions struct {
	BaseDir          string
	SkippedDir       string
	Overwrite        bool
	ExcludeKotsKinds bool
	IsHelmBase       bool
}

func (b *Base) WriteBase(options WriteOptions) error {
	renderDir := filepath.Join(options.BaseDir, b.Path)

	_, err := os.Stat(renderDir)
	if err == nil {
		if options.Overwrite {
			if err := os.RemoveAll(renderDir); err != nil {
				return errors.Wrap(err, "failed to remove previous content in base")
			}
		} else {
			return fmt.Errorf("directory %s already exists", renderDir)
		}
	}

	if _, _, err := b.writeBase(options, true); err != nil {
		return errors.Wrap(err, "failed to write root base")
	}

	if err := b.writeSkippedFiles(options); err != nil {
		return errors.Wrap(err, "failed to write skipped files")
	}

	return nil
}

func (b *Base) writeBase(options WriteOptions, isTopLevelBase bool) ([]string, []kustomizetypes.PatchStrategicMerge, error) {
	renderDir := filepath.Join(options.BaseDir, b.Path)

	if _, err := os.Stat(renderDir); os.IsNotExist(err) {
		if err := os.MkdirAll(renderDir, 0744); err != nil {
			return nil, nil, errors.Wrap(err, "failed to mkdir for base root")
		}
	}

	resources, patches, err := deduplicateOnContent(b.Files, options.ExcludeKotsKinds, b.Namespace)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to deduplicate content")
	}

	kustomizeResources := []string{}
	kustomizePatches := []kustomizetypes.PatchStrategicMerge{}
	kustomizeBases := []string{}

	for _, file := range resources {
		fileRenderPath := filepath.Join(renderDir, file.Path)
		d, _ := path.Split(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0744); err != nil {
				return nil, nil, errors.Wrap(err, "failed to mkdir")
			}
		}

		newContent, err := kotsutil.RemoveNilFieldsFromYAML(file.Content)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to remove empty mapping fields")
		}

		if err := ioutil.WriteFile(fileRenderPath, newContent, 0644); err != nil {
			return nil, nil, errors.Wrap(err, "failed to write base file")
		}

		kustomizeResources = append(kustomizeResources, path.Join(".", file.Path))
	}

	for _, file := range patches {
		fileRenderPath := filepath.Join(renderDir, file.Path)
		d, _ := path.Split(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0744); err != nil {
				return nil, nil, errors.Wrap(err, "failed to mkdir")
			}
		}

		if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
			return nil, nil, errors.Wrap(err, "failed to write base file")
		}

		kustomizePatches = append(kustomizePatches, kustomizetypes.PatchStrategicMerge(path.Join(".", file.Path)))
	}

	// Additional files are not included in the kustomization.yaml
	for _, file := range b.AdditionalFiles {
		fileRenderPath := filepath.Join(renderDir, file.Path)
		d, _ := path.Split(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0744); err != nil {
				return nil, nil, errors.Wrap(err, "failed to mkdir")
			}
		}

		if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
			return nil, nil, errors.Wrap(err, "failed to write additional file")
		}
	}

	subResources := []string{}
	subPatches := []kustomizetypes.PatchStrategicMerge{}
	for _, base := range b.Bases {
		if base.Path == "" {
			return nil, nil, errors.New("kustomize sub-base path cannot be empty")
		}
		options := WriteOptions{
			BaseDir:          filepath.Join(options.BaseDir, b.Path),
			Overwrite:        options.Overwrite,
			ExcludeKotsKinds: options.ExcludeKotsKinds,
		}
		baseResources, basePatches, err := base.writeBase(options, false)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to render base %s", base.Path)
		}
		if base.Namespace == "" {
			for _, r := range baseResources {
				subResources = append(subResources, filepath.Join(base.Path, r))
			}
			for _, p := range basePatches {
				subPatches = append(subPatches, kustomizetypes.PatchStrategicMerge(filepath.Join(base.Path, string(p))))
			}
		} else {
			kustomizeBases = append(kustomizeBases, base.Path)
		}
	}

	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		MetaData: &kustomizetypes.ObjectMeta{
			Annotations: map[string]string{
				"kots.io/kustomization": "base",
			},
		},
		Namespace:             b.Namespace,
		Resources:             kustomizeResources,
		PatchesStrategicMerge: kustomizePatches,
		Bases:                 kustomizeBases,
	}

	if isTopLevelBase && !options.IsHelmBase {
		// For the top level base, the one that isn't a helm chart), "bases" should contain all charts that are deployed to different namespaces.
		// "resources" will then be deduplicated and split into "resources" and "patches", where patches will contain duplicate resources.
		// This is done for backwards compatibility with apps that include duplicate resources in different bases.
		resources, patches, err := deduplicateResources(append(kustomization.Resources, subResources...), options.BaseDir, options.ExcludeKotsKinds, b.Namespace)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to defuplicate top level kustomize")
		}
		kustomization.Resources = resources
		kustomization.PatchesStrategicMerge = append(kustomization.PatchesStrategicMerge, patches...)
	}

	if err := k8sutil.WriteKustomizationToFile(kustomization, filepath.Join(renderDir, "kustomization.yaml")); err != nil {
		return nil, nil, errors.Wrap(err, "failed to write kustomization to file")
	}

	kustomizeResources = append(kustomizeResources, subResources...)
	kustomizePatches = append(kustomizePatches, subPatches...)
	return kustomizeResources, kustomizePatches, nil
}

type SkippedFilesIndex struct {
	SkippedFiles []SkippedFile `yaml:"skippedFiles"`
}

type SkippedFile struct {
	Path   string `yaml:"path"`
	Reason string `yaml:"reason"`
}

func (b *Base) writeSkippedFiles(options WriteOptions) error {
	// if we dont render this dir we will get an error when we create the archive
	renderDir := filepath.Join(options.SkippedDir, b.Path)

	_, err := os.Stat(renderDir)
	if err == nil {
		if options.Overwrite {
			if err := os.RemoveAll(renderDir); err != nil {
				return errors.Wrap(err, "failed to remove previous content in skipped files")
			}
		} else {
			return fmt.Errorf("directory %s already exists", renderDir)
		}
	}

	if _, err := os.Stat(renderDir); os.IsNotExist(err) {
		if err := os.MkdirAll(renderDir, 0744); err != nil {
			return errors.Wrap(err, "failed to mkdir for skipped files root")
		}
	}

	errorFiles := b.getErrorFiles()

	if len(errorFiles) == 0 {
		return nil
	}

	index := SkippedFilesIndex{SkippedFiles: []SkippedFile{}}
	for _, file := range errorFiles {
		fileRenderPath := filepath.Join(renderDir, file.Path)
		d := path.Dir(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0744); err != nil {
				return errors.Wrap(err, "failed to mkdir")
			}
		}

		if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
			return errors.Wrapf(err, "failed to write skipped file %s", fileRenderPath)
		}

		index.SkippedFiles = append(index.SkippedFiles, SkippedFile{
			Path:   file.Path,
			Reason: fmt.Sprintf("%v", file.Error),
		})
	}

	indexOut, err := yaml.Marshal(index)
	if err != nil {
		return errors.Wrap(err, "failed to marshal skipped files index")
	}
	fileRenderPath := filepath.Join(renderDir, "_index.yaml")
	if err := ioutil.WriteFile(fileRenderPath, indexOut, 0644); err != nil {
		return errors.Wrap(err, "failed to write skipped files index")
	}

	return nil
}

func (b *Base) getErrorFiles() []BaseFile {
	errorFiles := b.ErrorFiles
	for _, base := range b.Bases {
		baseErrorFiles := []BaseFile{}
		for _, errorFile := range base.getErrorFiles() {
			baseErrorFiles = append(baseErrorFiles, BaseFile{
				Path:    path.Join(base.Path, errorFile.Path),
				Content: errorFile.Content,
				Error:   errorFile.Error,
			})
		}
		errorFiles = append(errorFiles, baseErrorFiles...)
	}
	return errorFiles
}

func deduplicateResources(filePaths []string, baseDir string, excludeKotsKinds bool, baseNS string) ([]string, []kustomizetypes.PatchStrategicMerge, error) {
	files := []BaseFile{}
	for _, filePath := range filePaths {
		content, err := ioutil.ReadFile(filepath.Join(baseDir, filePath))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to read base file %s", filePath)
		}

		files = append(files, BaseFile{Path: filePath, Content: content})
	}

	resourcesFiles, patchFiles, err := deduplicateOnContent(files, excludeKotsKinds, baseNS)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to deduplicate base file")
	}

	resources := []string{}
	for _, f := range resourcesFiles {
		resources = append(resources, f.Path)
	}

	patches := []kustomizetypes.PatchStrategicMerge{}
	for _, f := range patchFiles {
		patches = append(patches, kustomizetypes.PatchStrategicMerge(f.Path))
	}

	return resources, patches, nil
}

func deduplicateOnContent(files []BaseFile, excludeKotsKinds bool, baseNS string) ([]BaseFile, []BaseFile, error) {
	resources := []BaseFile{}
	patches := []BaseFile{}

	foundGVKNamesMap := map[string]bool{}

	singleDocs := convertToSingleDocBaseFiles(files)

	for _, file := range singleDocs {
		writeToKustomization, err := file.ShouldBeIncludedInBaseKustomization(excludeKotsKinds)
		if err != nil {
			// should we do anything with errors here?
			if _, ok := err.(ParseError); !ok {
				return nil, nil, errors.Wrap(err, "failed to check if file should be included")
			}
		}

		if writeToKustomization {
			thisGVKName, _ := GetGVKWithNameAndNs(file.Content, baseNS)
			found := foundGVKNamesMap[thisGVKName]

			if !found || thisGVKName == "" {
				resources = append(resources, file)
				foundGVKNamesMap[thisGVKName] = true
			} else {
				patches = append(patches, file)
			}
		}
	}

	return resources, patches, nil
}

func convertToSingleDocBaseFiles(files []BaseFile) []BaseFile {
	singleDocs := []BaseFile{}
	for _, file := range files {
		docs := util.ConvertToSingleDocs(file.Content)
		// This is here so as not to change previous behavior
		if len(docs) == 0 {
			singleDocs = append(singleDocs, BaseFile{
				Path:    file.Path,
				Content: []byte(""),
			})
			continue
		}
		for idx, doc := range docs {
			filename := file.Path
			if idx > 0 {
				filename = strings.TrimSuffix(file.Path, filepath.Ext(file.Path))
				filename = fmt.Sprintf("%s-%d%s", filename, idx+1, filepath.Ext(file.Path))
			}

			baseFile := BaseFile{
				Path:    filename,
				Content: doc,
			}

			singleDocs = append(singleDocs, baseFile)
		}
	}
	return singleDocs
}
