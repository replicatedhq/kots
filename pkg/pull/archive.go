package pull

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/base"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func writeArchiveAsConfigMap(pullOptions PullOptions, u *upstreamtypes.Upstream, baseDir string) error {
	// Package this app into a bundle so that the Admin Console can write it as the first version...
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}

	paths := []string{
		path.Join(pullOptions.RootDir, u.Name, "upstream"),
		path.Join(pullOptions.RootDir, u.Name, "base"),
		path.Join(pullOptions.RootDir, u.Name, "overlays"),
	}

	renderedPath := path.Join(pullOptions.RootDir, "rendered")
	if _, err := os.Stat(renderedPath); err == nil {
		paths = append(paths, renderedPath)
	}

	skippedFilesPath := path.Join(pullOptions.RootDir, "skippedFiles")
	if _, err := os.Stat(skippedFilesPath); err == nil {
		paths = append(paths, skippedFilesPath)
	}

	tempDir, err := ioutil.TempDir("", "kots")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tempDir)

	if err := tarGz.Archive(paths, path.Join(tempDir, "kots-uploadable-archive.tar.gz")); err != nil {
		return errors.Wrap(err, "failed to create tar gz")
	}

	archive, err := ioutil.ReadFile(path.Join(tempDir, "kots-uploadable-archive.tar.gz"))
	if err != nil {
		return errors.Wrap(err, "failed to read temp file")
	}

	encoded := base64.StdEncoding.EncodeToString(archive)

	// well.
	//
	// let's write this encoded value to a config map with a known label
	// so that kotsadm will upload it and ingest it as an app
	// it's really the only way we can get the archive
	// but etcd and config maps are limited to 1 mb
	// so let's split it across multiple, if it's larger than 1 mb
	// 768*1024 was chosen as a number sufficiently below 1m so as to allow
	// for padding or other inefficiencies.
	encodedParts, err := util.SplitStringOnLen(encoded, 768*1024)
	if err != nil {
		return errors.Wrap(err, "failed to split encoded")
	}

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	for i, encodedPart := range encodedParts {
		labels := map[string]string{}
		labels["kotsadm"] = "bundle"
		labels["kotsadm-bundle-part"] = strconv.Itoa(i)

		configMap := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   fmt.Sprintf("kotsadm-bundle-%d", i),
				Labels: labels,
			},
			Data: map[string]string{
				"part": encodedPart,
			},
		}

		var b bytes.Buffer
		if err := s.Encode(configMap, &b); err != nil {
			return errors.Wrap(err, "failed to marshal bundle part config map")
		}

		if err := base.AddBundlePart(baseDir, fmt.Sprintf("kotsadm-bundle-%d.yaml", i), b.Bytes()); err != nil {
			return errors.Wrap(err, "failed to write base")
		}
	}

	return nil
}

func CleanBaseArchive(path string) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return errors.Wrap(err, "failed to read dir")
	}

	// "overlays" contains manual kustomizations.
	// "upstream" contains config values, known images, and other important installation info
	// "kotsKinds" contains the rendered kotskinds which can be needed when processing incoming updates
	// everything else should be deleted and generated again
	for _, file := range files {
		switch file.Name() {
		case "overlays", "upstream", "kotsKinds":
			continue
		default:
			err := os.RemoveAll(filepath.Join(path, file.Name()))
			if err != nil {
				return errors.Wrapf(err, "failed to delete %s", file.Name())
			}
		}
	}

	return nil
}
