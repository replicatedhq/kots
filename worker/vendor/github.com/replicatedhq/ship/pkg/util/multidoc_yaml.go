package util

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v3"
	"sigs.k8s.io/kustomize/pkg/types"
)

type outputYaml struct {
	name     string
	contents string
}

// this function is not perfect, and has known limitations. One of these is that it does not account for `\n---\n` in multiline strings.
func MaybeSplitMultidocYaml(ctx context.Context, fs afero.Afero, localPath string) error {
	files, err := fs.ReadDir(localPath)
	if err != nil {
		return errors.Wrapf(err, "read files in %s", localPath)
	}

	for _, file := range files {
		if file.IsDir() {
			if err := MaybeSplitMultidocYaml(ctx, fs, filepath.Join(localPath, file.Name())); err != nil {
				return err
			}
		}

		if filepath.Ext(file.Name()) != ".yaml" && filepath.Ext(file.Name()) != ".yml" {
			// not yaml, nothing to do
			continue
		}

		inFileBytes, err := fs.ReadFile(filepath.Join(localPath, file.Name()))
		if err != nil {
			return errors.Wrapf(err, "read %s", filepath.Join(localPath, file.Name()))
		}

		outputFiles := []outputYaml{}
		filesStrings := strings.Split(string(inFileBytes), "\n---\n")
		crds := []string{}

		// generate replacement yaml files
		for idx, fileString := range filesStrings {

			newOutputFiles, newCRDs, err := generateOutputYaml(idx, fileString)
			if err != nil {
				return errors.Wrapf(err, "at path %s", file.Name())
			}

			outputFiles = append(outputFiles, newOutputFiles...)
			crds = append(crds, newCRDs...)
		}

		if len(crds) > 0 {
			crdsFile := outputYaml{contents: strings.Join(crds, "\n---\n"), name: "CustomResourceDefinitions"}
			outputFiles = append(outputFiles, crdsFile)
		}

		if len(outputFiles) < 2 {
			// not a multidoc yaml, or at least not a multidoc kubernetes yaml
			continue
		}

		// delete multidoc yaml file
		err = fs.Remove(filepath.Join(localPath, file.Name()))
		if err != nil {
			return errors.Wrapf(err, "unable to remove %s", filepath.Join(localPath, file.Name()))
		}

		// write replacement yaml
		for _, outputFile := range outputFiles {
			err = fs.WriteFile(filepath.Join(localPath, outputFile.name+".yaml"), []byte(outputFile.contents), os.FileMode(0644))
			if err != nil {
				return errors.Wrapf(err, "write %s", outputFile.name)
			}
		}
	}

	return nil
}

// this function drops files with no parsable 'kind', separates out CRD definitions, and splits list yaml into multiple files
func generateOutputYaml(idx int, fileString string) ([]outputYaml, []string, error) {

	thisOutputFile := outputYaml{contents: fileString}
	theseOutputFiles := []outputYaml{}
	crds := []string{}

	thisMetadata := MinimalK8sYaml{}
	_ = yaml.Unmarshal([]byte(fileString), &thisMetadata)

	if thisMetadata.Kind == "" {
		// ignore invalid k8s yaml
		return nil, nil, nil
	}

	if thisMetadata.Kind == "CustomResourceDefinition" {
		// collate CRDs into one file
		crds = append(crds, fileString)
		return theseOutputFiles, crds, nil
	}

	if thisMetadata.Kind == "List" {
		// split list yaml into multiple files
		thisList := ListK8sYaml{}
		_ = yaml.Unmarshal([]byte(fileString), &thisList)

		for itemIdx, item := range thisList.Items {
			itemYaml, err := MarshalIndent(2, item)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "marshal item %d from file %d", itemIdx, idx)
			}

			newOutput, newCRDs, err := generateOutputYaml(itemIdx, string(itemYaml))
			if err != nil {
				return nil, nil, errors.Wrapf(err, "at file %d", idx)
			}

			theseOutputFiles = append(theseOutputFiles, newOutput...)
			crds = append(crds, newCRDs...)
		}

		return theseOutputFiles, crds, nil
	}

	fileName := GenerateNameFromMetadata(thisMetadata, idx)
	thisOutputFile.name = fileName

	theseOutputFiles = []outputYaml{thisOutputFile}
	return theseOutputFiles, crds, nil
}

// SplitAllKustomize ensures that all yaml files within the designated directory are split into one resource per file, excepting CRDs
// if a kustomization yaml existed beforehand, it rewrites the resource list to match the new filenames
// if not, it creates a kustomization yaml
// if an existing kustomization yaml referred to other bases, splitAll will be called on those bases as well
func SplitAllKustomize(fs afero.Afero, path string) error {
	existingKustomize, err := fs.Exists(filepath.Join(path, "kustomization.yaml"))
	if err != nil {
		return errors.Wrapf(err, "find kustomization yaml in %s", path)
	}

	if existingKustomize {
		// kustomize yaml exists, so read it in and go from there
		err = splitKustomizeDir(fs, path)
		if err != nil {
			return err
		}
	} else {
		// kustomize yaml does not exist, so split yaml files and generate a kustomization yaml from the results
		err = splitYamlDir(fs, path)
		if err != nil {
			return err
		}
	}

	return nil
}

// split kustomize resources and patches inside a directory that already contains a kustomization.yaml
func splitKustomizeDir(fs afero.Afero, path string) error {
	kustomizeYaml, err := fs.ReadFile(filepath.Join(path, "kustomization.yaml"))
	if err != nil {
		return errors.Wrapf(err, "read kustomization yaml in %s", path)
	}

	kustomization := types.Kustomization{}
	err = yaml.Unmarshal(kustomizeYaml, &kustomization)
	if err != nil {
		return errors.Wrapf(err, "unmarshal kustomization yaml from %s", path)
	}

	// run the 'split into one k8s yaml per file' process on each base this depends on
	for _, newBase := range kustomization.Bases {
		err = SplitAllKustomize(fs, filepath.Join(path, newBase))
		if err != nil {
			return errors.Wrapf(err, "split base %s of %s", newBase, path)
		}
	}

	newResources := []string{}
	// split every k8s resource this kustomization yaml depends on
	for _, resourcePath := range kustomization.Resources {
		stat, err := fs.Stat(filepath.Join(path, resourcePath))
		if err != nil {
			return errors.Wrapf(err, "stat resource yaml at %s", filepath.Join(path, resourcePath))
		}
		inFileBytes, err := fs.ReadFile(filepath.Join(path, resourcePath))
		if err != nil {
			return errors.Wrapf(err, "read resource yaml at %s", filepath.Join(path, resourcePath))
		}

		filesStrings := strings.Split(string(inFileBytes), "\n---\n")
		validFileStrings := []string{}
		validMetadatas := []MinimalK8sYaml{}

		for _, fileString := range filesStrings {
			// check if the file is valid k8s yaml
			// if it is, add it to the list
			// if it is not, discard it
			thisMetadata := MinimalK8sYaml{}
			_ = yaml.Unmarshal([]byte(fileString), &thisMetadata)

			if thisMetadata.Kind == "" || thisMetadata.Metadata.Name == "" {
				continue
			} else {
				validFileStrings = append(validFileStrings, fileString)
				validMetadatas = append(validMetadatas, thisMetadata)
			}
		}

		if len(validFileStrings) == 1 {
			// if there is only one valid yaml in this set of strings, there is no need to rename anything
			err = fs.WriteFile(filepath.Join(path, resourcePath), []byte(validFileStrings[0]), stat.Mode())
			if err != nil {
				return errors.Wrapf(err, "write updated k8s resource at %s", filepath.Join(path, resourcePath))
			}
			newResources = append(newResources, resourcePath)
			continue
		}

		if len(validFileStrings) == 0 {
			// ???
			// there should be at least one
			// for now let kustomize handle it until something breaks
			newResources = append(newResources, resourcePath)
			continue
		}

		if len(validFileStrings) > 1 {
			// we need to do some renaming, since there were multiple files in this resource
			for idx, fileString := range validFileStrings {
				thisMetadata := validMetadatas[idx]
				fileName := GenerateNameFromMetadata(thisMetadata, idx) + ".yaml"
				newPath := filepath.Join(path, filepath.Dir(resourcePath), fileName)
				err = fs.WriteFile(newPath, []byte(fileString), stat.Mode())
				if err != nil {
					return errors.Wrapf(err, "write split k8s resource at %s", newPath)
				}
				newResources = append(newResources, filepath.Join(filepath.Dir(resourcePath), fileName))
			}

			// we also need to remove the original
			err = fs.Remove(filepath.Join(path, resourcePath))
			if err != nil {
				return errors.Wrapf(err, "remove replaced multidoc file %s", filepath.Join(path, resourcePath))
			}
		}
	}
	kustomization.Resources = newResources

	// split every k8s strategic merge this depends on
	for _, strategicMerge := range kustomization.PatchesStrategicMerge {
		_, err := fs.ReadFile(filepath.Join(path, string(strategicMerge)))
		if err != nil {
			return errors.Wrapf(err, "read strategicMerge yaml at %s", filepath.Join(path, string(strategicMerge)))
		}
		// TODO actually split this yaml and edit kustomization
	}

	kustYamlBytes, err := MarshalIndent(2, kustomization)
	if err != nil {
		return errors.Wrapf(err, "marshal edited kustomization yaml from %s", filepath.Join(path, "kustomization.yaml"))
	}

	err = fs.WriteFile(filepath.Join(path, "kustomization.yaml"), kustYamlBytes, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "write edited kustomization yaml to %s", filepath.Join(path, "kustomization.yaml"))
	}

	return nil
}

// split kubernetes resources inside a directory with no kustomization.yaml and create a kustomization.yaml for the results
func splitYamlDir(fs afero.Afero, path string) error {
	// split kubernetes resources
	err := MaybeSplitMultidocYaml(context.Background(), fs, path)
	if err != nil {
		return errors.Wrapf(err, "split yaml dir %s", path)
	}

	// generate kustomization yaml for kubernetes resources
	err = generateKustomizationYaml(fs, path)
	if err != nil {
		return errors.Wrapf(err, "generate kustomization yaml for %s", path)
	}

	return nil
}

// given a dir containing k8s yaml and no kustomization yaml, make a kustomization yaml containing all the k8s yaml as resources
func generateKustomizationYaml(fs afero.Afero, path string) error {
	kustomization := types.Kustomization{}
	dirFiles := []string{}
	err := fs.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			dirFiles = append(dirFiles, path)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "generate kustomization yaml for %s", path)
	}

	for _, dirFile := range dirFiles {
		if ShouldAddFileToBase(&fs, []string{}, dirFile) {
			relPath, err := filepath.Rel(path, dirFile)
			if err != nil {
				return errors.Wrapf(err, "get relative path to file %s from %s to generate kustomization", dirFile, path)
			}
			kustomization.Resources = append(kustomization.Resources, relPath)
		}
	}

	kustomizationBytes, err := MarshalIndent(2, kustomization)
	if err != nil {
		return errors.Wrapf(err, "marshal kustomization yaml for %s", path)
	}

	err = fs.WriteFile(filepath.Join(path, "kustomization.yaml"), kustomizationBytes, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "write kustomization yaml for %s", path)
	}

	return nil
}

// RecursiveNormalizeCopyKustomize copies kubernetes yaml from the source directory into the dest directory.
// `kustomization.yaml` files are skipped.
// if `kustomization.yaml` contains any bases, those bases are recursively copied into `destDir/<base-name>/`
func RecursiveNormalizeCopyKustomize(fs afero.Afero, sourceDir, destDir string) error {
	err := RecursiveCopy(fs, sourceDir, destDir)
	if err != nil {
		return errors.Wrapf(err, "normalize and copy %s to %s", sourceDir, destDir)
	}

	kustExists, err := fs.Exists(filepath.Join(sourceDir, "kustomization.yaml"))
	if err != nil {
		return errors.Wrapf(err, "check if kustomization exists in %s", sourceDir)
	}
	if !kustExists {
		// no other bases to copy, and no patches/kustomization to filter
		return nil
	}

	kustBytes, err := fs.ReadFile(filepath.Join(sourceDir, "kustomization.yaml"))
	if err != nil {
		return errors.Wrapf(err, "read kustomization in %s", sourceDir)
	}

	kustomization := types.Kustomization{}
	err = yaml.Unmarshal(kustBytes, &kustomization)
	if err != nil {
		return errors.Wrapf(err, "parse kustomization in %s", sourceDir)
	}

	// remove strategic merge patches from the copied files - their contents will be added to the relevant bases
	for _, patch := range kustomization.PatchesStrategicMerge {
		patchString := string(patch)
		err = fs.Remove(filepath.Join(destDir, patchString))
		if err != nil {
			return errors.Wrapf(err, "remove patch at %s from destDir %s when copying from %s", patchString, destDir, sourceDir)
		}
	}

	for _, newBase := range kustomization.Bases {
		newBase = filepath.Clean(filepath.Join(sourceDir, newBase))
		cleanBase := strings.ReplaceAll(newBase, string(filepath.Separator), "-")
		// cleanBase = "base-" + cleanBase
		err = RecursiveNormalizeCopyKustomize(fs, newBase, filepath.Join(destDir, cleanBase))
		if err != nil {
			return errors.Wrapf(err, "copy base %s of %s", newBase, sourceDir)
		}
	}

	err = fs.Remove(filepath.Join(destDir, "kustomization.yaml"))
	if err != nil {
		return errors.Wrapf(err, "remove kustomization from rendered yaml dir %s", destDir)
	}

	return nil
}
