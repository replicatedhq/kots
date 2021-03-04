package kotsadm

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/containers/image/v5/docker/tarfile"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	imageslogs "github.com/google/go-containerregistry/pkg/logs"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func PushImages(airgapArchive string, options types.PushImagesOptions) error {
	imagesRootDir, err := ioutil.TempDir("", "kotsadm-airgap")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(imagesRootDir)

	err = ExtractAirgapImages(airgapArchive, imagesRootDir, options.ProgressWriter)
	if err != nil {
		return errors.Wrap(err, "failed to extract images")
	}

	err = pushKotsadmImagesFromPath(imagesRootDir, options)
	if err != nil {
		return errors.Wrap(err, "failed to list image formats")
	}

	return nil
}

func ExtractAirgapImages(archive string, destDir string, progressWriter io.Writer) error {
	reader, err := os.Open(archive)
	if err != nil {
		return errors.Wrap(err, "failed to open airgap archive")
	}
	defer reader.Close()

	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return errors.Wrap(err, "failed to get new gzip reader")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return errors.Wrap(err, "failed to read tar header")
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		dstFileName := filepath.Join(destDir, header.Name)
		if err := os.MkdirAll(filepath.Dir(dstFileName), 0755); err != nil {
			return errors.Wrap(err, "failed to create path")
		}

		err = func() error {
			writeProgressLine(progressWriter, fmt.Sprintf("Extracting %s", dstFileName))

			dstFile, err := os.Create(dstFileName)
			if err != nil {
				return errors.Wrap(err, "failed to create file")
			}
			defer dstFile.Close()

			if _, err := io.Copy(dstFile, tarReader); err != nil {
				return errors.Wrap(err, "failed to copy file data")
			}

			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

func pushKotsadmImagesFromPath(rootDir string, options types.PushImagesOptions) error {
	fileInfos, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return errors.Wrap(err, "failed to read dir")
	}

	for _, info := range fileInfos {
		if !info.IsDir() {
			continue
		}

		err = processImageNames(rootDir, info.Name(), options)
		if err != nil {
			return errors.Wrapf(err, "failed list images names for format %s", info.Name())
		}
	}

	return nil
}

func processImageNames(rootDir string, format string, options types.PushImagesOptions) error {
	fileInfos, err := ioutil.ReadDir(filepath.Join(rootDir, format))
	if err != nil {
		return errors.Wrap(err, "failed to read dir")
	}

	for _, info := range fileInfos {
		if !info.IsDir() {
			continue
		}

		err = processImageTags(rootDir, format, info.Name(), options)
		if err != nil {
			return errors.Wrapf(err, "failed list tags for image %s", info.Name())
		}
	}

	return nil
}

func processImageTags(rootDir string, format string, imageName string, options types.PushImagesOptions) error {
	fileInfos, err := ioutil.ReadDir(filepath.Join(rootDir, format, imageName))
	if err != nil {
		return errors.Wrap(err, "failed to read dir")
	}

	for _, info := range fileInfos {
		if info.IsDir() {
			continue
		}

		err = pushOneImage(rootDir, format, imageName, info.Name(), options)
		if err != nil {
			return errors.Wrapf(err, "failed push image %s:%s", imageName, info.Name())
		}
	}

	return nil
}

func pushOneImage(rootDir string, format string, imageName string, tag string, options types.PushImagesOptions) error {
	prevProgress := imageslogs.Progress
	defer func() {
		imageslogs.Progress = prevProgress
	}()

	if options.ProgressWriter != nil {
		imageslogs.Progress = log.New(options.ProgressWriter, "", log.LstdFlags)
	}

	craneOptions := []crane.Option{
		crane.Insecure,
	}

	if options.Registry.Username != "" && options.Registry.Password != "" {
		authConfig := authn.AuthConfig{
			Username: options.Registry.Username,
			Password: options.Registry.Password,
		}
		craneOptions = append(craneOptions, crane.WithAuth(authn.FromConfig(authConfig)))
	}

	destStr := fmt.Sprintf("%s/%s:%s", options.Registry.Endpoint, imageName, tag)
	writeProgressLine(options.ProgressWriter, fmt.Sprintf("Pushing %s", destStr))

	imageFile := filepath.Join(rootDir, format, imageName, tag)
	imageReader, err := image.RegistryImageFromReader(imageFile)
	if err != nil {
		return errors.Wrap(err, "failed to create image reader 1")
	}

	err = image.PushImageFromStream(imageReader, destStr, craneOptions)
	if err != nil {
		return errors.Wrapf(err, "failed to push image")
	}

	return nil
}

func writeProgressLine(progressWriter io.Writer, line string) {
	fmt.Fprint(progressWriter, fmt.Sprintf("%s\n", line))
}

func TagAndPushAppImagesFromPath(imagesDir string, options types.PushImagesOptions) ([]kustomizetypes.Image, error) {
	formatDirs, err := ioutil.ReadDir(imagesDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read images dir")
	}

	imageFiles := make(map[string]*types.ImageFile)
	images := []kustomizetypes.Image{}
	for _, f := range formatDirs {
		if !f.IsDir() {
			continue
		}

		formatRoot := path.Join(imagesDir, f.Name())
		err := filepath.Walk(formatRoot,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				layers := make(map[string]*types.LayerInfo)
				if options.LogForUI {
					layers, err = getLayerInfo(path)
					if err != nil {
						return errors.Wrap(err, "failed to get layer info")
					}
				}

				imageFiles[path] = &types.ImageFile{
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

		reportWriter := options.ProgressWriter
		imageslogs.Progress = log.New(reportWriter, "", log.LstdFlags)
		if options.LogForUI {
			wc := reportWriterWithProgress(imageFiles, options.ProgressWriter)
			reportWriter = wc.(io.Writer)
			defer wc.Write([]byte(fmt.Sprintf("+status.flush:\n")))
			defer wc.Close()
		}

		for _, imageFile := range imageFiles {
			formatRoot := path.Join(imagesDir, imageFile.Format)
			pathWithoutRoot := imageFile.FilePath[len(formatRoot)+1:]
			rewrittenImage, err := image.ImageInfoFromFile(options.Registry, strings.Split(pathWithoutRoot, string(os.PathSeparator)))
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode image from path")
			}

			if options.LogForUI {
				// still log in console for future reference
				fmt.Printf("Pushing image %s:%s\n", rewrittenImage.NewName, rewrittenImage.NewTag)
			} else {
				writeProgressLine(reportWriter, fmt.Sprintf("Pushing image %s:%s", rewrittenImage.NewName, rewrittenImage.NewTag))
			}

			registryAuth := image.RegistryAuth{
				Username: options.Registry.Username,
				Password: options.Registry.Password,
			}

			imageFile.UploadStart = time.Now()
			if options.LogForUI {
				reportWriter.Write([]byte(fmt.Sprintf("+file.begin:%s\n", imageFile.FilePath)))
			}
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
				if options.LogForUI {
					reportWriter.Write([]byte(fmt.Sprintf("+file.error:%s\n", err)))
				}
				options.Log.FinishChildSpinner()
				return nil, errors.Wrap(err, "failed to copy file to registry")
			}

			options.Log.FinishChildSpinner()

			imageFile.UploadEnd = time.Now()
			if options.LogForUI {
				reportWriter.Write([]byte(fmt.Sprintf("+file.end:%s\n", imageFile.FilePath)))
			}

			images = append(images, rewrittenImage)
		}
	}

	return images, nil
}

func TagAndPushAppImagesFromBundle(airgapBundle string, options types.PushImagesOptions) ([]kustomizetypes.Image, error) {
	if options.LogForUI {
		writeProgressLine(options.ProgressWriter, "Reading image information from bundle...")
	}

	imageFiles, err := getImageListFromBundle(airgapBundle, options.LogForUI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get layer info from bundle")
	}

	fileReader, err := os.Open(airgapBundle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer fileReader.Close()

	gzipReader, err := gzip.NewReader(fileReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get new gzip reader")
	}
	defer gzipReader.Close()

	reportWriter := options.ProgressWriter
	imageslogs.Progress = log.New(reportWriter, "", log.LstdFlags)
	if options.LogForUI {
		wc := reportWriterWithProgress(imageFiles, options.ProgressWriter)
		reportWriter = wc.(io.Writer)
		defer wc.Write([]byte(fmt.Sprintf("+status.flush:\n")))
		defer wc.Close()
	}

	images := []kustomizetypes.Image{}

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to get read archive")
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		imageFile, ok := imageFiles[header.Name]
		if !ok {
			continue
		}

		err = func() error {
			if options.LogForUI {
				reportWriter.Write([]byte(fmt.Sprintf("+file.begin:%s\n", imageFile.FilePath)))
			}

			pathParts := strings.Split(imageFile.FilePath, string(os.PathSeparator))
			if len(pathParts) < 3 {
				return errors.Errorf("not enough path parts in %q", imageFile.FilePath)
			}

			rewrittenImage, err := image.ImageInfoFromFile(options.Registry, pathParts[2:])
			if err != nil {
				return errors.Wrap(err, "failed to decode image from path")
			}

			if options.LogForUI {
				writeProgressLine(reportWriter, fmt.Sprintf("Extracting image %s:%s", rewrittenImage.NewName, rewrittenImage.NewTag))
			}

			tmpFile, err := ioutil.TempFile("", "kotsadm-app-image-")
			if err != nil {
				return errors.Wrap(err, "failed to create temp file")
			}
			defer tmpFile.Close()
			defer os.Remove(tmpFile.Name())

			gzipWriter := gzip.NewWriter(tmpFile)
			_, err = io.Copy(gzipWriter, tarReader)
			if err != nil {
				return errors.Wrapf(err, "failed to write file %q", header.Name)
			}

			// Close file to flush all data before pushing to registry
			if err := gzipWriter.Close(); err != nil {
				return errors.Wrap(err, "failed to close gzip writer")
			}
			if err := tmpFile.Close(); err != nil {
				return errors.Wrap(err, "failed to close tmp file")
			}

			if options.LogForUI {
				// still log in console for future reference
				fmt.Printf("Pushing image %s:%s\n", rewrittenImage.NewName, rewrittenImage.NewTag)
			} else {
				writeProgressLine(reportWriter, fmt.Sprintf("Pushing image %s:%s", rewrittenImage.NewName, rewrittenImage.NewTag))
			}

			registryAuth := image.RegistryAuth{
				Username: options.Registry.Username,
				Password: options.Registry.Password,
			}

			imageFile.UploadStart = time.Now()
			for i := 0; i < 5; i++ {
				err = image.CopyFromFileToRegistry(tmpFile.Name(), rewrittenImage.NewName, rewrittenImage.NewTag, rewrittenImage.Digest, registryAuth, reportWriter)
				if err == nil {
					break // image copy succeeded, exit the retry loop
				} else {
					options.Log.ChildActionWithoutSpinner("encountered error (#%d) copying image, waiting 10s before trying again: %s", i+1, err.Error())
					time.Sleep(time.Second * 10)
				}
			}
			if err != nil {
				if options.LogForUI {
					reportWriter.Write([]byte(fmt.Sprintf("+file.error:%s\n", err)))
				}
				options.Log.FinishChildSpinner()
				return errors.Wrap(err, "failed to copy file from bundle to registry")
			}

			options.Log.FinishChildSpinner()

			imageFile.UploadEnd = time.Now()
			if options.LogForUI {
				reportWriter.Write([]byte(fmt.Sprintf("+file.end:%s\n", imageFile.FilePath)))
			}

			images = append(images, rewrittenImage)

			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return images, nil
}

func getImageListFromBundle(airgapBundle string, getLayerInfo bool) (map[string]*types.ImageFile, error) {
	fileReader, err := os.Open(airgapBundle)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer fileReader.Close()

	gzipReader, err := gzip.NewReader(fileReader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get new gzip reader")
	}
	defer gzipReader.Close()

	imageFiles := make(map[string]*types.ImageFile)

	tarReader := tar.NewReader(gzipReader)
	foundImagesFolder := false
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to get read archive")
		}

		// Airgap bundle will have some small files in the beginning.
		// The rest of it will be images in folders.
		if !foundImagesFolder {
			if header.Name == "." {
				continue
			}
			if header.Typeflag == tar.TypeReg {
				continue
			}
			foundImagesFolder = true
			continue
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		layers := make(map[string]*types.LayerInfo)
		if getLayerInfo {
			layers, err = getLayerInfoFromReader(tarReader)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get layer info")
			}
		}

		pathParts := strings.Split(header.Name, string(os.PathSeparator))
		if len(pathParts) < 3 {
			return nil, errors.Errorf("not enough parts in image path: %q", header.Name)
		}

		imageFiles[header.Name] = &types.ImageFile{
			Format:   pathParts[1], // path is like "images/<format>/image/name/tag"
			FilePath: header.Name,
			Layers:   layers,
			FileSize: header.Size,
			Status:   "queued",
		}
	}
	return imageFiles, nil
}

func getLayerInfo(path string) (map[string]*types.LayerInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open image archive")
	}
	defer f.Close()
	return getLayerInfoFromReader(f)
}

func getLayerInfoFromReader(reader io.Reader) (map[string]*types.LayerInfo, error) {
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

		layers := make(map[string]*types.LayerInfo)
		for _, l := range manifestItems[0].Layers {
			fileInfo, found := files[l]
			if !found {
				return nil, errors.Errorf("layer %s not found in tar archive", l)
			}

			id := strings.TrimSuffix(l, ".tar")
			layer := &types.LayerInfo{
				ID:   id,
				Size: fileInfo.Size,
			}
			layers[id] = layer
		}
		return layers, nil
	}

	return nil, errors.New("manifest.json not found")
}

func reportWriterWithProgress(files map[string]*types.ImageFile, reportWriter io.Writer) io.WriteCloser {
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		currentFilePath := ""
		currentLine := ""

		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			line := scanner.Text()
			// Example sequence of messages we get per image

			// 2021/03/03 20:47:26 Copying from nginx to ttl.sh/ns/nginx:latest
			// 2021/03/03 20:47:29 existing blob: sha256:35c43ace9216212c0f0e546a65eec93fa9fc8e96b25880ee222b7ed2ca1d2151
			// 2021/03/03 20:47:31 pushed blob: sha256:f5a38c5f8d4e817a6d0fdc705abc21677c15ad68ab177500e4e34b70e02a201b
			// 2021/03/03 20:47:32 pushed blob: sha256:ec3bd7de90d781b1d3e3a55fc40b1ec332b591360fb62dd10b8f28799c2297c1
			// 2021/03/03 20:47:32 pushed blob: sha256:19e2441aeeab2ac2e850795573c62b9aad2c302e126a34ed370ad46ab91e6218
			// 2021/03/03 20:47:32 pushed blob: sha256:83500d85111837bbc4a04125fd930f68067e4de851a56d89bd2e03cc3bf7e8ca
			// 2021/03/03 20:47:35 pushed blob: sha256:8acc495f1d914a74439c21bf43c4319672e0f4ba51f9cfafa042a1051ef52671
			// 2021/03/03 20:47:35 pushed blob: sha256:45b42c59be334ecda0daaa139b2f7d310e45c564c5f12263b1b8e68ec9e810ed
			// 2021/03/03 20:47:36 ttl.sh/ns/nginx@sha256:b08ecc9f7997452ef24358f3e43b9c66888fadb31f3e5de22fec922975caa75a: digest: sha256:b08ecc9f7997452ef24358f3e43b9c66888fadb31f3e5de22fec922975caa75a size: 1570

			timePrefixLen := len("YYYY/MM/MM HH:MM:SS")
			timePrefix := line[:timePrefixLen]
			if _, err := time.Parse("2006/01/02 15:04:05", timePrefix); err == nil {
				line = line[timePrefixLen+1:]
			}

			if strings.HasPrefix(line, "existing blob:") {
				currentLine = line
				progressLayerEnded(currentFilePath, files)
				writeCurrentProgress(currentLine, currentFilePath, files, reportWriter)
				continue
			} else if strings.HasPrefix(line, "pushed blob:") {
				currentLine = line
				progressLayerEnded(currentFilePath, files)
				writeCurrentProgress(currentLine, currentFilePath, files, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+file.begin:") {
				currentFilePath = strings.TrimPrefix(line, "+file.begin:")
				progressFileStarted(currentFilePath, files)
				writeCurrentProgress(currentLine, currentFilePath, files, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+file.end:") {
				progressFileEnded(currentFilePath, files)
				writeCurrentProgress(currentLine, currentFilePath, files, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+file.error:") {
				errorStr := strings.TrimPrefix(line, "+file.error:")
				progressFileFailed(currentFilePath, files, errorStr)
				writeCurrentProgress(currentLine, currentFilePath, files, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+status.flush:") {
				writeCurrentProgress(currentLine, currentFilePath, files, reportWriter)
				continue
			} else {
				currentLine = line
				writeCurrentProgress(currentLine, currentFilePath, files, reportWriter)
				continue
			}
		}
	}()

	return pipeWriter
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

func progressLayerEnded(filePath string, files map[string]*types.ImageFile) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "uploading"
	file.LayersUploaded++
}

func progressLayerStarted(filePath string, files map[string]*types.ImageFile) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "uploading"
}

func progressFileStarted(filePath string, files map[string]*types.ImageFile) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "uploading"
	file.UploadStart = time.Now()
}

func progressFileEnded(filePath string, files map[string]*types.ImageFile) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "uploaded"
	file.UploadEnd = time.Now()
}

func progressFileFailed(filePath string, files map[string]*types.ImageFile, errorStr string) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "failed"
	file.Error = errorStr
	file.UploadEnd = time.Now()
}

func writeCurrentProgress(line string, filePath string, files map[string]*types.ImageFile, reportWriter io.Writer) {
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
			Current:     file.LayersUploaded,
			Total:       int64(len(file.Layers)) + 1, // crane.Push reports 1 extra blob than the number of layers in the manifest
			StartTime:   file.UploadStart,
			EndTime:     file.UploadEnd,
		}

		// Just in case, since crane can report extra blobs
		if progressImage.Current > progressImage.Total {
			progressImage.Current = progressImage.Total
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
