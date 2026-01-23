package archive

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"go.podman.io/image/v5/docker/tarfile"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/types"
)

func GetImageLayers(path string) ([]types.Layer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open image archive")
	}
	defer f.Close()
	return GetImageLayersFromReader(f)
}

func GetImageLayersFromReader(reader io.Reader) ([]types.Layer, error) {
	tarReader := tar.NewReader(reader)

	var manifestItems []tarfile.ManifestItem
	files := make(map[string]*tar.Header)
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to advance in tar archive")
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		files[header.Name] = header
		if header.Name != "manifest.json" {
			continue
		}

		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(tarReader)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read manifest from tar archive")
		}

		if err := json.Unmarshal(buf.Bytes(), &manifestItems); err != nil {
			return nil, errors.Wrap(err, "failed to decode manifest.json")
		}

		if len(manifestItems) != 1 {
			return nil, errors.Errorf("manifest.json: expected 1 item, got %d", len(manifestItems))
		}

		layers := []types.Layer{}
		for _, l := range manifestItems[0].Layers {
			fileInfo, found := files[l]
			if !found {
				return nil, errors.Errorf("layer %s not found in tar archive", l)
			}
			layer := types.Layer{
				Digest: fmt.Sprintf("sha256:%s", strings.TrimSuffix(l, ".tar")),
				Size:   fileInfo.Size,
			}
			layers = append(layers, layer)
		}
		return layers, nil
	}

	return nil, errors.New("manifest.json not found")
}
