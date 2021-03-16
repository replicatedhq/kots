package kotsadm

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker/tarfile"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	containerstypes "github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"k8s.io/client-go/kubernetes/scheme"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func PushImages(airgapArchive string, options types.PushImagesOptions) error {
	imagesRootDir, err := ioutil.TempDir("", "kotsadm-airgap")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(imagesRootDir)

	err = ExtractAppAirgapArchive(airgapArchive, imagesRootDir, false, options.ProgressWriter)
	if err != nil {
		return errors.Wrap(err, "failed to extract images")
	}

	if isAppArchive(imagesRootDir) {
		_, err := TagAndPushAppImagesFromPath(filepath.Join(imagesRootDir, "images"), options)
		if err != nil {
			return errors.Wrap(err, "failed to push app images")
		}
	} else {
		err = pushKotsadmImagesFromPath(imagesRootDir, options)
		if err != nil {
			return errors.Wrap(err, "failed to push kotsadm images")
		}
	}

	return nil
}

func ExtractAppAirgapArchive(archive string, destDir string, excludeImages bool, progressWriter io.Writer) error {
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

		if header.Name == "." {
			continue
		}

		if err != nil {
			return errors.Wrap(err, "failed to read tar header")
		}

		if excludeImages && header.Typeflag == tar.TypeDir {
			// Once we hit a directory, the rest of the archive is images.
			break
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
	var imagePolicy = []byte(`{
		"default": [{"type": "insecureAcceptAnything"}]
	  }`)

	policy, err := signature.NewPolicyFromBytes(imagePolicy)
	if err != nil {
		return errors.Wrap(err, "failed to read default policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return errors.Wrap(err, "failed to create policy")
	}

	destCtx := &containerstypes.SystemContext{
		DockerInsecureSkipTLSVerify: containerstypes.OptionalBoolTrue,
		DockerDisableV1Ping:         true,
	}
	if options.Registry.Username != "" && options.Registry.Password != "" {
		destCtx.DockerAuthConfig = &containerstypes.DockerAuthConfig{
			Username: options.Registry.Username,
			Password: options.Registry.Password,
		}
	}
	if os.Getenv("KOTSADM_INSECURE_SRCREGISTRY") == "true" {
		// allow pulling images from http/invalid https docker repos
		// intended for development only, _THIS MAKES THINGS INSECURE_
		destCtx.DockerInsecureSkipTLSVerify = containerstypes.OptionalBoolTrue
	}

	dstTag := tag
	if options.KotsadmTag != "" {
		dstTag = options.KotsadmTag
	}

	destStr := fmt.Sprintf("%s/%s:%s", options.Registry.Endpoint, imageName, dstTag)
	destRef, err := alltransports.ParseImageName(fmt.Sprintf("docker://%s", destStr))
	if err != nil {
		return errors.Wrapf(err, "failed to parse dest image name %s", destStr)
	}

	imageFile := filepath.Join(rootDir, format, imageName, tag)
	localRef, err := alltransports.ParseImageName(fmt.Sprintf("%s:%s", format, imageFile))
	if err != nil {
		return errors.Wrapf(err, "failed to parse local image name: %s:%s", format, imageFile)
	}

	writeProgressLine(options.ProgressWriter, fmt.Sprintf("Pushing %s", destStr))

	_, err = image.CopyImageWithGC(context.Background(), policyContext, destRef, localRef, &copy.Options{
		RemoveSignatures:      true,
		SignBy:                "",
		ReportWriter:          options.ProgressWriter,
		SourceCtx:             nil,
		DestinationCtx:        destCtx,
		ForceManifestMIMEType: "",
	})
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
				return nil, errors.Wrap(err, "failed to push image")
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

			_, err = io.Copy(tmpFile, tarReader)
			if err != nil {
				return errors.Wrapf(err, "failed to write file %q", header.Name)
			}

			// Close file to flush all data before pushing to registry
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
			if options.LogForUI {
				reportWriter.Write([]byte(fmt.Sprintf("+file.begin:%s\n", imageFile.FilePath)))
			}
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
				return errors.Wrap(err, "failed to push image")
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

func GetImagesFromBundle(airgapBundle string, options types.PushImagesOptions) ([]kustomizetypes.Image, error) {
	if options.LogForUI {
		writeProgressLine(options.ProgressWriter, "Reading image information from bundle...")
	}

	imageFiles, err := getImageListFromBundle(airgapBundle, options.LogForUI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get layer info from bundle")
	}

	images := []kustomizetypes.Image{}
	for _, imageFile := range imageFiles {
		pathParts := strings.Split(imageFile.FilePath, string(os.PathSeparator))
		if len(pathParts) < 3 {
			return nil, errors.Errorf("not enough path parts in %q", imageFile.FilePath)
		}

		rewrittenImage, err := image.ImageInfoFromFile(options.Registry, pathParts[2:])
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode image from path")
		}

		images = append(images, rewrittenImage)
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

func progressLayerEnded(filePath, layerID string, files map[string]*types.ImageFile) {
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

func progressLayerStarted(filePath, layerID string, files map[string]*types.ImageFile) {
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

func progressFileStarted(filePath, layerID string, files map[string]*types.ImageFile) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "uploading"
	file.UploadStart = time.Now()
}

func progressFileEnded(filePath, layerID string, files map[string]*types.ImageFile) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "uploaded"
	file.UploadEnd = time.Now()
}

func progressFileFailed(filePath, layerID string, files map[string]*types.ImageFile, errorStr string) {
	file := files[filePath]
	if file == nil {
		return
	}

	file.Status = "failed"
	file.Error = errorStr
	file.UploadEnd = time.Now()
}

func writeCurrentProgress(line, filePath, layerID string, files map[string]*types.ImageFile, reportWriter io.Writer) {
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

func countLayersUploaded(image *types.ImageFile) int64 {
	count := int64(0)
	for _, layer := range image.Layers {
		if !layer.UploadEnd.IsZero() {
			count += 1
		}
	}
	return count
}

func isAppArchive(rootDir string) bool {
	fileInfos, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return false
	}

	for _, info := range fileInfos {
		if info.IsDir() || filepath.Ext(info.Name()) != ".yaml" {
			continue
		}

		contents, err := ioutil.ReadFile(filepath.Join(rootDir, info.Name()))
		if err != nil {
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		_, gvk, err := decode(contents, nil, nil)
		if err != nil {
			continue
		}

		if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Airgap" {
			return true
		}
	}

	return false
}
