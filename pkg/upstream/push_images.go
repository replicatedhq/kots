package upstream

import (
	"archive/tar"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/containers/image/v5/docker/tarfile"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type PushUpstreamImageOptions struct {
	RootDir             string
	ImagesDir           string
	CreateAppDir        bool
	Log                 *logger.Logger
	ReplicatedRegistry  registry.RegistryOptions
	ReportWriter        io.Writer
	DestinationRegistry registry.RegistryOptions
}

type ImageFile struct {
	Format      string
	Status      string
	Error       string
	FilePath    string
	Layers      map[string]*LayerInfo
	FileSize    int64
	UploadStart time.Time
	UploadEnd   time.Time
}

type LayerInfo struct {
	ID          string
	Size        int64
	UploadStart time.Time
	UploadEnd   time.Time
}

func TagAndPushUpstreamImages(u *types.Upstream, options PushUpstreamImageOptions) ([]kustomizetypes.Image, error) {
	formatDirs, err := ioutil.ReadDir(options.ImagesDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read images dir")
	}

	imageFiles := make(map[string]*ImageFile)
	images := []kustomizetypes.Image{}
	for _, f := range formatDirs {
		if !f.IsDir() {
			continue
		}

		formatRoot := path.Join(options.ImagesDir, f.Name())
		err := filepath.Walk(formatRoot,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				layers, err := getLayerInfo(path)
				if err != nil {
					return errors.Wrap(err, "failed to get layer info")
				}

				imageFiles[path] = &ImageFile{
					Format:   f.Name(),
					FilePath: path,
					Layers:   layers,
					FileSize: info.Size(),
					Status:   "queued",
				}
				return nil
			})
		if err != nil {
			return nil, errors.Wrap(err, "failed to walk images dir")
		}

		reportWriter := reportWriterWithProgress(imageFiles, options.ReportWriter)
		defer reportWriter.Write([]byte(fmt.Sprintf("+status.flush:\n")))
		defer reportWriter.Close()

		for _, imageFile := range imageFiles {
			formatRoot := path.Join(options.ImagesDir, imageFile.Format)
			pathWithoutRoot := imageFile.FilePath[len(formatRoot)+1:]
			rewrittenImage, err := image.ImageInfoFromFile(options.DestinationRegistry, strings.Split(pathWithoutRoot, string(os.PathSeparator)))
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode image from path")
			}

			// copy to the registry
			options.Log.ChildActionWithSpinner("Pushing image %s:%s", rewrittenImage.NewName, rewrittenImage.NewTag)

			registryAuth := image.RegistryAuth{
				Username: options.DestinationRegistry.Username,
				Password: options.DestinationRegistry.Password,
			}

			imageFile.UploadStart = time.Now()
			reportWriter.Write([]byte(fmt.Sprintf("+file.begin:%s\n", imageFile.FilePath)))
			for i := 0; i < 5; i++ {
				err = image.CopyFromFileToRegistry(imageFile.FilePath, rewrittenImage.NewName, rewrittenImage.NewTag, rewrittenImage.Digest, registryAuth, reportWriter)
				if err == nil {
					break // image copy succeeded, exit the retry loop
				} else {
					options.Log.ChildActionWithoutSpinner("encountered error (#%d) copying image, waiting 10s before trying again: %s", i+1, err.Error())
					time.Sleep(time.Second * 10)
				}
			}
			if err != nil {
				reportWriter.Write([]byte(fmt.Sprintf("+file.error:%s\n", err)))
				options.Log.FinishChildSpinner()
				return nil, errors.Wrap(err, "failed to push image")
			}

			options.Log.FinishChildSpinner()
			imageFile.UploadEnd = time.Now()
			reportWriter.Write([]byte(fmt.Sprintf("+file.end:%s\n", imageFile.FilePath)))

			images = append(images, buildImageAltNames(rewrittenImage)...)
		}
	}

	return images, nil
}

func buildImageAltNames(rewrittenImage kustomizetypes.Image) []kustomizetypes.Image {
	// kustomize does string based comparison, so all of these are treated as different images:
	// docker.io/library/redis:latest
	// redis:latest
	// redis
	// As a workaround we add all 3 to the list

	// similarly, docker.io/notlibrary/image:tag needs to be rewritten
	// as notlibrary/image:tag (and the same handling for 'latest')

	images := []kustomizetypes.Image{rewrittenImage}

	rewrittenName := rewrittenImage.Name
	if strings.HasPrefix(rewrittenName, "docker.io/library/") {
		rewrittenName = strings.TrimPrefix(rewrittenName, "docker.io/library/")
		images = append(images, kustomizetypes.Image{
			Name:    rewrittenName,
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
	} else if strings.HasPrefix(rewrittenName, "docker.io/") {
		rewrittenName = strings.TrimPrefix(rewrittenName, "docker.io/")
		images = append(images, kustomizetypes.Image{
			Name:    rewrittenName,
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})

	}

	if strings.HasSuffix(rewrittenName, ":latest") {
		rewrittenName = strings.TrimSuffix(rewrittenName, ":latest")
		images = append(images, kustomizetypes.Image{
			Name:    rewrittenName,
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
	}

	return images
}

func reportWriterWithProgress(files map[string]*ImageFile, reportWriter io.Writer) io.WriteCloser {
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		currentLayerID := ""
		currentFilePath := ""
		currentLine := ""

		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			line := scanner.Text()
			// Example sequence of messages we get per image
			//
			// Copying blob sha256:67cddc63a0c4a6dd25d2c7789f7b7cdd9ce1a5d05a0607303c0ef625d0b76d08
			// Copying blob sha256:5dacd731af1b0386ead06c8b1feff9f65d9e0bdfec032d2cd0bc03690698feda
			// Copying blob sha256:b66a10934ed6942a31f8d0e96b1646fe0cbc7a9e0dd58eb686585d3e2d2edd1b
			// Copying blob sha256:0e401eb4a60a193c933bf80ebeab0ac35ac2592bc7c048d6843efb6b1d2f593a
			// Copying config sha256:043316b7542bc66eb4dad30afb998086714862c863f0f267467385fada943681
			// Writing manifest to image destination
			// Storing signatures

			if strings.HasPrefix(line, "Copying blob sha256:") {
				currentLine = line
				progressLayerEnded(currentFilePath, currentLayerID, files)
				currentLayerID = strings.TrimSuffix(strings.TrimPrefix(line, "Copying blob sha256:"), ".tar")
				progressLayerStarted(currentFilePath, currentLayerID, files)
				writeCurrentProgress(currentLine, currentFilePath, currentLayerID, files, reportWriter)
				continue
			} else if strings.HasPrefix(line, "Copying config sha256:") {
				currentLine = line
				progressLayerEnded(currentFilePath, currentLayerID, files)
				writeCurrentProgress(currentLine, currentFilePath, currentLayerID, files, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+file.begin:") {
				currentFilePath = strings.TrimPrefix(line, "+file.begin:")
				progressFileStarted(currentFilePath, currentLayerID, files)
				writeCurrentProgress(currentLine, currentFilePath, currentLayerID, files, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+file.end:") {
				progressFileEnded(currentFilePath, currentLayerID, files)
				writeCurrentProgress(currentLine, currentFilePath, currentLayerID, files, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+file.error:") {
				errorStr := strings.TrimPrefix(line, "+file.error:")
				progressFileFailed(currentFilePath, currentLayerID, files, errorStr)
				writeCurrentProgress(currentLine, currentFilePath, currentLayerID, files, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+status.flush:") {
				writeCurrentProgress(currentLine, currentFilePath, currentLayerID, files, reportWriter)
				continue
			} else {
				currentLine = line
				writeCurrentProgress(currentLine, currentFilePath, currentLayerID, files, reportWriter)
				continue
			}
		}
	}()

	return pipeWriter
}

func getLayerInfo(path string) (map[string]*LayerInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open image archive")
	}
	defer f.Close()

	tarReader := tar.NewReader(f)

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
	}

	layers := make(map[string]*LayerInfo)
	for _, l := range manifestItems[0].Layers {
		fileInfo, found := files[l]
		if !found {
			return nil, errors.Errorf("layer %s not found in tar archive %s", l, path)
		}

		id := strings.TrimSuffix(l, ".tar")
		layer := &LayerInfo{
			ID:   id,
			Size: fileInfo.Size,
		}
		layers[id] = layer
	}

	return layers, nil
}

type ProgressReport struct {
	// set to "progressReport"
	Type string `json:"type"`
	// the same progress text that used to be sent in unstructured message
	CompatibilityMessage string `json:"compatibilityMessage"`
	// all images found in archive
	Images []ProgressImage `json:"images"`
}

type ProgressImage struct {
	// image name and tag, "nginx:latest"
	DisplayName string `json:"displayName"`
	// image upload status: queued, uploading, uploaded, failed
	Status string `json:"status"`
	// error string set when status is failed
	Error string `json:"error"`
	// amount currently uploaded (currently number of layers)
	Current int64 `json:"current"`
	// total amount that needs to be uploaded (currently number of layers)
	Total int64 `json:"total"`
	// time when image started uploading
	StartTime time.Time `json:"startTime"`
	// time when image finished uploading
	EndTime time.Time `json:"endTime"`
}

func progressLayerEnded(filePath, layerID string, files map[string]*ImageFile) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "uploading"

	layer := file.Layers[layerID]
	if layer == nil {
		return
	}

	layer.UploadEnd = time.Now()
}

func progressLayerStarted(filePath, layerID string, files map[string]*ImageFile) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "uploading"

	layer := file.Layers[layerID]
	if layer == nil {
		return
	}

	layer.UploadStart = time.Now()
}

func progressFileStarted(filePath, layerID string, files map[string]*ImageFile) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "uploading"
	file.UploadStart = time.Now()
}

func progressFileEnded(filePath, layerID string, files map[string]*ImageFile) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "uploaded"
	file.UploadEnd = time.Now()
}

func progressFileFailed(filePath, layerID string, files map[string]*ImageFile, errorStr string) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "failed"
	file.Error = errorStr
	file.UploadEnd = time.Now()
}

func writeCurrentProgress(line, filePath, layerID string, files map[string]*ImageFile, reportWriter io.Writer) {
	report := ProgressReport{
		Type:                 "progressReport",
		CompatibilityMessage: line,
	}

	images := make([]ProgressImage, 0)
	for path, file := range files {
		progressImage := ProgressImage{
			DisplayName: pathToDisplayName(path),
			Status:      file.Status,
			Error:       file.Error,
			Current:     countLayersUploaded(file),
			Total:       int64(len(file.Layers)),
			StartTime:   file.UploadStart,
			EndTime:     file.UploadEnd,
		}
		images = append(images, progressImage)
	}
	report.Images = images
	data, _ := json.Marshal(report)
	fmt.Fprintf(reportWriter, "%s\n", data)
}

func pathToDisplayName(path string) string {
	tag := filepath.Base(path)
	image := filepath.Base(filepath.Dir(path))
	return image + ":" + tag // TODO: support for SHAs
}

func countLayersUploaded(image *ImageFile) int64 {
	count := int64(0)
	for _, layer := range image.Layers {
		if !layer.UploadEnd.IsZero() {
			count += 1
		}
	}
	return count
}
