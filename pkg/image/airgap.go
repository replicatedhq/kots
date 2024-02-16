package image

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/containers/image/v5/transports/alltransports"
	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/archives"
	dockerarchive "github.com/replicatedhq/kots/pkg/docker/archive"
	dockerregistry "github.com/replicatedhq/kots/pkg/docker/registry"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	dockertypes "github.com/replicatedhq/kots/pkg/docker/types"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/imageutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
)

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
		if err != nil {
			return errors.Wrap(err, "failed to read tar header")
		}

		if header.Name == "." {
			continue
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
			WriteProgressLine(progressWriter, fmt.Sprintf("Extracting %s", dstFileName))

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

func WriteProgressLine(progressWriter io.Writer, line string) {
	fmt.Fprint(progressWriter, fmt.Sprintf("%s\n", line))
}

// CopyAirgapImages pushes images found in the airgap bundle/airgap root to the configured registry.
func CopyAirgapImages(opts imagetypes.ProcessImageOptions, log *logger.CLILogger) error {
	pushOpts := imagetypes.PushImagesOptions{
		Registry: dockerregistrytypes.RegistryOptions{
			Endpoint:  opts.RegistrySettings.Hostname,
			Namespace: opts.RegistrySettings.Namespace,
			Username:  opts.RegistrySettings.Username,
			Password:  opts.RegistrySettings.Password,
		},
		Log:            log,
		ProgressWriter: opts.ReportWriter,
		LogForUI:       true,
	}

	if opts.AirgapBundle != "" {
		err := TagAndPushImagesFromBundle(opts.AirgapBundle, pushOpts)
		if err != nil {
			return errors.Wrap(err, "failed to push images from bundle")
		}
	} else if opts.AirgapRoot != "" {
		err := TagAndPushImagesFromPath(opts.AirgapRoot, pushOpts)
		if err != nil {
			return errors.Wrap(err, "failed to push images from dir")
		}
	}

	return nil
}

func TagAndPushImagesFromPath(airgapRootDir string, options imagetypes.PushImagesOptions) error {
	airgap, err := kotsutil.FindAirgapMetaInDir(airgapRootDir)
	if err != nil {
		return errors.Wrap(err, "failed to find airgap meta")
	}

	switch airgap.Spec.Format {
	case dockertypes.FormatDockerRegistry:
		return PushImagesFromTempRegistry(airgapRootDir, airgap.Spec.SavedImages, options)
	case dockertypes.FormatDockerArchive, "":
		return PushImagesFromDockerArchivePath(airgapRootDir, options)
	default:
		return errors.Errorf("Airgap bundle format '%s' is not supported", airgap.Spec.Format)
	}
}

func TagAndPushImagesFromBundle(airgapBundle string, options imagetypes.PushImagesOptions) error {
	airgap, err := kotsutil.FindAirgapMetaInBundle(airgapBundle)
	if err != nil {
		return errors.Wrap(err, "failed to find airgap meta")
	}

	switch airgap.Spec.Format {
	case dockertypes.FormatDockerRegistry:
		extractedBundle, err := os.MkdirTemp("", "extracted-airgap-kots")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir for unarchived airgap bundle")
		}
		defer os.RemoveAll(extractedBundle)

		tarGz := archiver.TarGz{
			Tar: &archiver.Tar{
				ImplicitTopLevelFolder: false,
			},
		}
		if err := tarGz.Unarchive(airgapBundle, extractedBundle); err != nil {
			return errors.Wrap(err, "falied to unarchive airgap bundle")
		}
		return PushImagesFromTempRegistry(extractedBundle, airgap.Spec.SavedImages, options)
	case dockertypes.FormatDockerArchive, "":
		return PushImagesFromDockerArchiveBundle(airgapBundle, options)
	default:
		return errors.Errorf("Airgap bundle format '%s' is not supported", airgap.Spec.Format)
	}
}

func PushImagesFromTempRegistry(airgapRootDir string, imageList []string, options imagetypes.PushImagesOptions) error {
	imagesDir := filepath.Join(airgapRootDir, "images")
	if _, err := os.Stat(imagesDir); os.IsNotExist(err) {
		// this can either be because images were already pushed from the CLI, or it's a diff airgap bundle with no images
		return nil
	}

	tempRegistry := &dockerregistry.TempRegistry{}
	if err := tempRegistry.Start(imagesDir); err != nil {
		return errors.Wrap(err, "failed to start temp registry")
	}
	defer tempRegistry.Stop()

	imageInfos := make(map[string]*imagetypes.ImageInfo)
	for _, image := range imageList {
		layerInfo := make(map[string]*imagetypes.LayerInfo)
		if options.LogForUI {
			layers, err := tempRegistry.GetImageLayers(image)
			if err != nil {
				return errors.Wrapf(err, "failed to get image layers for %s", image)
			}
			layerInfo, err = layerInfoFromLayers(layers)
			if err != nil {
				return errors.Wrap(err, "failed to get layer info")
			}
		}
		imageInfos[image] = &imagetypes.ImageInfo{
			Format: dockertypes.FormatDockerRegistry,
			Layers: layerInfo,
			Status: "queued",
		}
	}

	reportWriter := options.ProgressWriter
	if options.LogForUI {
		wc := reportWriterWithProgress(imageInfos, options.ProgressWriter)
		reportWriter = wc.(io.Writer)
		defer wc.Write([]byte("+status.flush:\n"))
		defer wc.Close()
	}

	for imageID, imageInfo := range imageInfos {
		srcRef, err := tempRegistry.SrcRef(imageID)
		if err != nil {
			return errors.Wrapf(err, "failed to parse source image %s", imageID)
		}

		destImage, err := imageutil.DestImage(options.Registry, imageID)
		if err != nil {
			return errors.Wrapf(err, "failed to get destination image for %s", imageID)
		}

		if options.KotsadmTag != "" {
			// kotsadm tag is set, change the tag of the kotsadm and kotsadm-migrations images
			imageName := imageutil.GetImageName(destImage)
			if imageName == "kotsadm" || imageName == "kotsadm-migrations" {
				di, err := imageutil.ChangeImageTag(destImage, options.KotsadmTag)
				if err != nil {
					return errors.Wrap(err, "failed to change kotsadm dest image tag")
				}
				destImage = di
			}
		}

		destStr := fmt.Sprintf("docker://%s", destImage)
		destRef, err := alltransports.ParseImageName(destStr)
		if err != nil {
			return errors.Wrapf(err, "failed to parse dest image %s", destStr)
		}

		// copy all architecures available in the bundle.
		// this also handles kotsadm airgap bundles that have multi-arch images but are referenced by tag.
		copyAll := true

		pushImageOpts := imagetypes.PushImageOptions{
			ImageID:      imageID,
			ImageInfo:    imageInfo,
			Log:          options.Log,
			LogForUI:     options.LogForUI,
			ReportWriter: reportWriter,
			CopyImageOptions: imagetypes.CopyImageOptions{
				SrcRef:  srcRef,
				DestRef: destRef,
				DestAuth: imagetypes.RegistryAuth{
					Username: options.Registry.Username,
					Password: options.Registry.Password,
				},
				CopyAll:           copyAll,
				SrcDisableV1Ping:  true,
				SrcSkipTLSVerify:  true,
				DestDisableV1Ping: true,
				DestSkipTLSVerify: true,
				ReportWriter:      reportWriter,
			},
		}
		if err := pushImage(pushImageOpts); err != nil {
			return errors.Wrapf(err, "failed to push image %s", imageID)
		}
	}

	return nil
}

func PushImagesFromDockerArchivePath(airgapRootDir string, options imagetypes.PushImagesOptions) error {
	imagesDir := filepath.Join(airgapRootDir, "images")
	if _, err := os.Stat(imagesDir); os.IsNotExist(err) {
		// images were already pushed from the CLI
		return nil
	}

	imageInfos := make(map[string]*imagetypes.ImageInfo)

	walkErr := filepath.Walk(imagesDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			layerInfo := make(map[string]*imagetypes.LayerInfo)
			if options.LogForUI {
				layers, err := dockerarchive.GetImageLayers(path)
				if err != nil {
					return errors.Wrap(err, "failed to get image layers")
				}
				layerInfo, err = layerInfoFromLayers(layers)
				if err != nil {
					return errors.Wrap(err, "failed to get layer info")
				}
			}

			imageInfos[path] = &imagetypes.ImageInfo{
				Format: dockertypes.FormatDockerArchive,
				Layers: layerInfo,
				Status: "queued",
			}
			return nil
		})
	if walkErr != nil {
		return errors.Wrap(walkErr, "failed to walk images dir")
	}

	reportWriter := options.ProgressWriter
	if options.LogForUI {
		wc := reportWriterWithProgress(imageInfos, options.ProgressWriter)
		reportWriter = wc.(io.Writer)
		defer wc.Write([]byte("+status.flush:\n"))
		defer wc.Close()
	}

	for imagePath, imageInfo := range imageInfos {
		formatRoot := path.Join(imagesDir, imageInfo.Format)
		pathWithoutRoot := imagePath[len(formatRoot)+1:]
		rewrittenImage, err := imageutil.RewriteDockerArchiveImage(options.Registry, strings.Split(pathWithoutRoot, string(os.PathSeparator)))
		if err != nil {
			return errors.Wrap(err, "failed to rewrite docker archive image")
		}

		srcRef, err := alltransports.ParseImageName(fmt.Sprintf("%s:%s", dockertypes.FormatDockerArchive, imagePath))
		if err != nil {
			return errors.Wrap(err, "failed to parse src image name")
		}

		destStr := fmt.Sprintf("docker://%s", imageutil.DestImageFromKustomizeImage(rewrittenImage))
		destRef, err := alltransports.ParseImageName(destStr)
		if err != nil {
			return errors.Wrapf(err, "failed to parse dest image name %s", destStr)
		}

		pushImageOpts := imagetypes.PushImageOptions{
			ImageID:      imagePath,
			ImageInfo:    imageInfo,
			Log:          options.Log,
			LogForUI:     options.LogForUI,
			ReportWriter: reportWriter,
			CopyImageOptions: imagetypes.CopyImageOptions{
				SrcRef:  srcRef,
				DestRef: destRef,
				DestAuth: imagetypes.RegistryAuth{
					Username: options.Registry.Username,
					Password: options.Registry.Password,
				},
				CopyAll:           false, // docker-archive format does not support multi-arch images
				DestSkipTLSVerify: true,
				DestDisableV1Ping: true,
				ReportWriter:      reportWriter,
			},
		}
		if err := pushImage(pushImageOpts); err != nil {
			return errors.Wrapf(err, "failed to push image %s", imagePath)
		}
	}

	return nil
}

func PushImagesFromDockerArchiveBundle(airgapBundle string, options imagetypes.PushImagesOptions) error {
	if exists, err := archives.DirExistsInAirgap("images", airgapBundle); err != nil {
		return errors.Wrap(err, "failed to check if images dir exists in airgap bundle")
	} else if !exists {
		// images were already pushed from the CLI
		return nil
	}

	if options.LogForUI {
		WriteProgressLine(options.ProgressWriter, "Reading image information from bundle...")
	}

	imageInfos, err := getImageInfosFromBundle(airgapBundle, options.LogForUI)
	if err != nil {
		return errors.Wrap(err, "failed to get images info from bundle")
	}

	fileReader, err := os.Open(airgapBundle)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer fileReader.Close()

	gzipReader, err := gzip.NewReader(fileReader)
	if err != nil {
		return errors.Wrap(err, "failed to get new gzip reader")
	}
	defer gzipReader.Close()

	reportWriter := options.ProgressWriter
	if options.LogForUI {
		wc := reportWriterWithProgress(imageInfos, options.ProgressWriter)
		reportWriter = wc.(io.Writer)
		defer wc.Write([]byte("+status.flush:\n"))
		defer wc.Close()
	}

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to get read archive")
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		imagePath := header.Name
		imageInfo, ok := imageInfos[imagePath]
		if !ok {
			continue
		}

		if options.LogForUI {
			WriteProgressLine(reportWriter, fmt.Sprintf("Extracting image %s", imagePath))
		}

		tmpFile, err := os.CreateTemp("", "kotsadm-image-")
		if err != nil {
			return errors.Wrap(err, "failed to create temp file")
		}
		defer tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		_, err = io.Copy(tmpFile, tarReader)
		if err != nil {
			return errors.Wrapf(err, "failed to write file %q", imagePath)
		}

		// Close file to flush all data before pushing to registry
		if err := tmpFile.Close(); err != nil {
			return errors.Wrap(err, "failed to close tmp file")
		}

		pathParts := strings.Split(imagePath, string(os.PathSeparator))
		if len(pathParts) < 3 {
			return errors.Errorf("not enough path parts in %q", imagePath)
		}

		rewrittenImage, err := imageutil.RewriteDockerArchiveImage(options.Registry, pathParts[2:])
		if err != nil {
			return errors.Wrap(err, "failed to rewrite docker archive image")
		}

		srcRef, err := alltransports.ParseImageName(fmt.Sprintf("%s:%s", dockertypes.FormatDockerArchive, tmpFile.Name()))
		if err != nil {
			return errors.Wrap(err, "failed to parse src image name")
		}

		destStr := fmt.Sprintf("docker://%s", imageutil.DestImageFromKustomizeImage(rewrittenImage))
		destRef, err := alltransports.ParseImageName(destStr)
		if err != nil {
			return errors.Wrapf(err, "failed to parse dest image name %s", destStr)
		}

		pushImageOpts := imagetypes.PushImageOptions{
			ImageID:      imagePath,
			ImageInfo:    imageInfo,
			Log:          options.Log,
			LogForUI:     options.LogForUI,
			ReportWriter: reportWriter,
			CopyImageOptions: imagetypes.CopyImageOptions{
				SrcRef:  srcRef,
				DestRef: destRef,
				DestAuth: imagetypes.RegistryAuth{
					Username: options.Registry.Username,
					Password: options.Registry.Password,
				},
				CopyAll:           false, // docker-archive format does not support multi-arch images
				DestSkipTLSVerify: true,
				DestDisableV1Ping: true,
				ReportWriter:      reportWriter,
			},
		}
		if err := pushImage(pushImageOpts); err != nil {
			return errors.Wrapf(err, "failed to push image %s", imagePath)
		}
	}

	return nil
}

func pushImage(opts imagetypes.PushImageOptions) error {
	opts.ImageInfo.UploadStart = time.Now()
	if opts.LogForUI {
		fmt.Printf("Pushing image %s\n", opts.ImageID) // still log in console for future reference
		opts.ReportWriter.Write([]byte(fmt.Sprintf("+file.begin:%s\n", opts.ImageID)))
	} else {
		destImageStr := opts.CopyImageOptions.DestRef.DockerReference().String() // this is better for debugging from the cli than the image id
		WriteProgressLine(opts.ReportWriter, fmt.Sprintf("Pushing image %s", destImageStr))
	}

	var retryAttempts int = 5
	var copyError error

	for i := 0; i < retryAttempts; i++ {
		copyError = CopyImage(opts.CopyImageOptions)
		if copyError == nil {
			break // image copy succeeded, exit the retry loop
		} else {
			opts.Log.ChildActionWithoutSpinner("encountered error (#%d) copying image, waiting 10s before trying again: %s", i+1, copyError.Error())
			time.Sleep(time.Second * 10)
		}
	}
	if copyError != nil {
		if opts.LogForUI {
			opts.ReportWriter.Write([]byte(fmt.Sprintf("+file.error:%s\n", copyError)))
		}
		opts.Log.FinishChildSpinner()
		return errors.Wrap(copyError, "failed to push image")
	}

	opts.Log.FinishChildSpinner()
	opts.ImageInfo.UploadEnd = time.Now()
	if opts.LogForUI {
		opts.ReportWriter.Write([]byte(fmt.Sprintf("+file.end:%s\n", opts.ImageID)))
	}

	return nil
}

func getImageInfosFromBundle(airgapBundle string, getLayerInfo bool) (map[string]*imagetypes.ImageInfo, error) {
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

	imageInfos := make(map[string]*imagetypes.ImageInfo)

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

		layerInfo := make(map[string]*imagetypes.LayerInfo)
		if getLayerInfo {
			layers, err := dockerarchive.GetImageLayersFromReader(tarReader)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get image layers from reader")
			}
			layerInfo, err = layerInfoFromLayers(layers)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get layer info")
			}
		}

		pathParts := strings.Split(header.Name, string(os.PathSeparator))
		if len(pathParts) < 3 {
			return nil, errors.Errorf("not enough parts in image path: %q", header.Name)
		}

		imageInfos[header.Name] = &imagetypes.ImageInfo{
			Format: dockertypes.FormatDockerArchive,
			Layers: layerInfo,
			Status: "queued",
		}
	}
	return imageInfos, nil
}

func layerInfoFromLayers(layers []dockertypes.Layer) (map[string]*imagetypes.LayerInfo, error) {
	layerInfo := make(map[string]*imagetypes.LayerInfo)
	for _, layer := range layers {
		layerID := strings.TrimPrefix(layer.Digest, "sha256:")
		layerInfo[layerID] = &imagetypes.LayerInfo{
			ID:   layerID,
			Size: layer.Size,
		}
	}
	return layerInfo, nil
}

func reportWriterWithProgress(imageInfos map[string]*imagetypes.ImageInfo, reportWriter io.Writer) io.WriteCloser {
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		currentLayerID := ""
		currentImageID := ""
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
				progressLayerEnded(currentImageID, currentLayerID, imageInfos)
				currentLayerID = strings.TrimPrefix(line, "Copying blob sha256:")
				progressLayerStarted(currentImageID, currentLayerID, imageInfos)
				writeCurrentProgress(currentLine, imageInfos, reportWriter)
				continue
			} else if strings.HasPrefix(line, "Copying config sha256:") {
				currentLine = line
				progressLayerEnded(currentImageID, currentLayerID, imageInfos)
				writeCurrentProgress(currentLine, imageInfos, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+file.begin:") {
				currentImageID = strings.TrimPrefix(line, "+file.begin:")
				progressFileStarted(currentImageID, imageInfos)
				writeCurrentProgress(currentLine, imageInfos, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+file.end:") {
				progressFileEnded(currentImageID, imageInfos)
				writeCurrentProgress(currentLine, imageInfos, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+file.error:") {
				errorStr := strings.TrimPrefix(line, "+file.error:")
				progressFileFailed(currentImageID, imageInfos, errorStr)
				writeCurrentProgress(currentLine, imageInfos, reportWriter)
				continue
			} else if strings.HasPrefix(line, "+status.flush:") {
				writeCurrentProgress(currentLine, imageInfos, reportWriter)
				continue
			} else {
				currentLine = line
				writeCurrentProgress(currentLine, imageInfos, reportWriter)
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

func progressLayerEnded(imageID, layerID string, imageInfos map[string]*imagetypes.ImageInfo) {
	imageInfo := imageInfos[imageID]
	if imageInfo == nil {
		return
	}

	imageInfo.Status = "uploading"

	layer := imageInfo.Layers[layerID]
	if layer == nil {
		return
	}

	layer.UploadEnd = time.Now()
}

func progressLayerStarted(imageID, layerID string, imageInfos map[string]*imagetypes.ImageInfo) {
	imageInfo := imageInfos[imageID]
	if imageInfo == nil {
		return
	}

	imageInfo.Status = "uploading"

	layer := imageInfo.Layers[layerID]
	if layer == nil {
		return
	}

	layer.UploadStart = time.Now()
}

func progressFileStarted(imageID string, imageInfos map[string]*imagetypes.ImageInfo) {
	imageInfo := imageInfos[imageID]
	if imageInfo == nil {
		return
	}

	imageInfo.Status = "uploading"
	imageInfo.UploadStart = time.Now()
}

func progressFileEnded(imageID string, imageInfos map[string]*imagetypes.ImageInfo) {
	imageInfo := imageInfos[imageID]
	if imageInfo == nil {
		return
	}

	imageInfo.Status = "uploaded"
	imageInfo.UploadEnd = time.Now()
}

func progressFileFailed(imageID string, imageInfos map[string]*imagetypes.ImageInfo, errorStr string) {
	imageInfo := imageInfos[imageID]
	if imageInfo == nil {
		return
	}

	imageInfo.Status = "failed"
	imageInfo.Error = errorStr
	imageInfo.UploadEnd = time.Now()
}

func writeCurrentProgress(line string, imageInfos map[string]*imagetypes.ImageInfo, reportWriter io.Writer) {
	report := ProgressReport{
		Type:                 "progressReport",
		CompatibilityMessage: line,
	}

	images := make([]ProgressImage, 0)
	for id, imageInfo := range imageInfos {
		displayName := ""
		if imageInfo.Format == dockertypes.FormatDockerArchive {
			displayName = pathToDisplayName(id)
		} else {
			displayName = id
		}
		progressImage := ProgressImage{
			DisplayName: displayName,
			Status:      imageInfo.Status,
			Error:       imageInfo.Error,
			Current:     countLayersUploaded(imageInfo),
			Total:       int64(len(imageInfo.Layers)),
			StartTime:   imageInfo.UploadStart,
			EndTime:     imageInfo.UploadEnd,
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

func countLayersUploaded(image *imagetypes.ImageInfo) int64 {
	count := int64(0)
	for _, layer := range image.Layers {
		if !layer.UploadEnd.IsZero() {
			count += 1
		}
	}
	return count
}
