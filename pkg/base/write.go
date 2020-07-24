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
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type WriteOptions struct {
	BaseDir          string
	Overwrite        bool
	ExcludeKotsKinds bool
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

	if _, err := os.Stat(renderDir); os.IsNotExist(err) {
		if err := os.MkdirAll(renderDir, 0744); err != nil {
			return errors.Wrap(err, "failed to mkdir for base root")
		}
	}

	resources, patches, err := deduplicateOnContent(b.Files, options.ExcludeKotsKinds, b.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to deduplicate content")
	}

	kustomizeResources := []string{}
	kustomizePatches := []kustomizetypes.PatchStrategicMerge{}
	kustomizeBases := []string{}

	for _, file := range resources {
		fileRenderPath := path.Join(renderDir, file.Path)
		d, _ := path.Split(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0744); err != nil {
				return errors.Wrap(err, "failed to mkdir")
			}
		}

		if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
			return errors.Wrap(err, "failed to write base file")
		}

		kustomizeResources = append(kustomizeResources, path.Join(".", file.Path))
	}

	for _, file := range patches {
		fileRenderPath := path.Join(renderDir, file.Path)
		d, _ := path.Split(fileRenderPath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0744); err != nil {
				return errors.Wrap(err, "failed to mkdir")
			}
		}

		if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
			return errors.Wrap(err, "failed to write base file")
		}

		kustomizePatches = append(kustomizePatches, kustomizetypes.PatchStrategicMerge(path.Join(".", file.Path)))
	}

	for _, base := range b.Bases {
		if base.Path == "" {
			return errors.New("kustomize sub-base path cannot be empty")
		}
		if err := base.WriteBase(options); err != nil {
			return errors.Wrapf(err, "failed to render base %q", base.Path)
		}
		kustomizeBases = append(kustomizeBases, base.Path)
	}

	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		Resources:             kustomizeResources,
		PatchesStrategicMerge: kustomizePatches,
		Bases:                 kustomizeBases,
	}
	if b.Namespace != "" {
		kustomization.Namespace = b.Namespace
	}

	if err := k8sutil.WriteKustomizationToFile(&kustomization, path.Join(renderDir, "kustomization.yaml")); err != nil {
		return errors.Wrap(err, "failed to write kustomization to file")
	}

	return nil
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

		if !writeToKustomization {
			continue
		}

		if writeToKustomization {
			thisGVKName := GetGVKWithNameAndNs(file.Content, baseNS)
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
		docs := convertToSingleDocs(file.Content)
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

func convertToSingleDocs(doc []byte) [][]byte {
	singleDocs := [][]byte{}
	docs := bytes.Split(doc, []byte("\n---\n"))
	for _, doc := range docs {
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}
		singleDocs = append(singleDocs, doc)
	}
	return singleDocs
}

func (b *Base) GetOverlaysDir(options WriteOptions) string {
	renderDir := options.BaseDir

	return path.Join(renderDir, "..", "overlays")
}
