package kotsadm

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	containerstypes "github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
)

func PushImages(options types.PushKotsadmImagesOptions) error {
	imagesRootDir, err := ioutil.TempDir("", "kotsadm-airgap")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(imagesRootDir)

	err = extractAirgapImages(options.AirgapArchive, imagesRootDir, options.ProgressWriter)
	if err != nil {
		return errors.Wrap(err, "failed to extract images")
	}

	err = readImageFormats(imagesRootDir, options)
	if err != nil {
		return errors.Wrap(err, "failed to list image formats")
	}

	return nil
}

func extractAirgapImages(archive string, destDir string, progressWriter io.Writer) error {
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

func readImageFormats(rootDir string, options types.PushKotsadmImagesOptions) error {
	fileInfos, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return errors.Wrap(err, "failed to read dir")
	}

	for _, info := range fileInfos {
		if !info.IsDir() {
			continue
		}

		err = readImageNames(rootDir, info.Name(), options)
		if err != nil {
			return errors.Wrapf(err, "failed list images names for format %s", info.Name())
		}
	}

	return nil
}

func readImageNames(rootDir string, format string, options types.PushKotsadmImagesOptions) error {
	fileInfos, err := ioutil.ReadDir(filepath.Join(rootDir, format))
	if err != nil {
		return errors.Wrap(err, "failed to read dir")
	}

	for _, info := range fileInfos {
		if !info.IsDir() {
			continue
		}

		err = readImageTags(rootDir, format, info.Name(), options)
		if err != nil {
			return errors.Wrapf(err, "failed list tags for image %s", info.Name())
		}
	}

	return nil
}

func readImageTags(rootDir string, format string, imageName string, options types.PushKotsadmImagesOptions) error {
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

func pushOneImage(rootDir string, format string, imageName string, tag string, options types.PushKotsadmImagesOptions) error {
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
	}
	if options.Username != "" && options.Password != "" {
		destCtx.DockerAuthConfig = &containerstypes.DockerAuthConfig{
			Username: options.Username,
			Password: options.Password,
		}
	}
	if os.Getenv("KOTSADM_INSECURE_SRCREGISTRY") == "true" {
		// allow pulling images from http/invalid https docker repos
		// intended for development only, _THIS MAKES THINGS INSECURE_
		destCtx.DockerInsecureSkipTLSVerify = containerstypes.OptionalBoolTrue
	}

	destStr := fmt.Sprintf("%s/%s:%s", options.Registry, imageName, tag)
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

	_, err = copy.Image(context.Background(), policyContext, destRef, localRef, &copy.Options{
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
