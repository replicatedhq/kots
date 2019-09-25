package main

import "C"

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/pull"
	"k8s.io/client-go/kubernetes/scheme"
)

//export MakeHelmChart
func MakeHelmChart(socket, upstreamData string, licenseData string, outputFile string) {
	go func() {
		var ffiResult *FFIResult

		statusClient, err := connectToStatusServer(socket)
		if err != nil {
			fmt.Printf("failed to connect to status server: %s\n", err)
			return
		}
		defer func() {
			statusClient.end(ffiResult)
		}()

		kotsscheme.AddToScheme(scheme.Scheme)
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(licenseData), nil, nil)
		if err != nil {
			fmt.Printf("failed to decode license data: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		license := obj.(*kotsv1beta1.License)

		licenseFile, err := ioutil.TempFile("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp file: %s\n", err)
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		defer os.Remove(licenseFile.Name())

		if err := ioutil.WriteFile(licenseFile.Name(), []byte(licenseData), 0644); err != nil {
			fmt.Printf("failed to write license to temp file: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		// pull to a tmp dir
		tmpRoot, err := ioutil.TempDir("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp root path: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}
		defer os.RemoveAll(tmpRoot)

		upstreamPath, err := ioutil.TempDir("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp upstream path: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		// TODO decompress the upstream data to upstream path

		pullOptions := pull.PullOptions{
			Downstreams:         []string{},
			LicenseFile:         licenseFile.Name(),
			ExcludeKotsKinds:    true,
			RootDir:             tmpRoot,
			ExcludeAdminConsole: true,
			CreateAppDir:        false,
			LocalPath:           upstreamPath,
		}

		renderDir, err := pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions)
		if err != nil {
			fmt.Printf("failed to pull upstream: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		outDir, err := ioutil.TempDir("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp outpuf dir: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		makeHelmChartOptions := helm.MakeHelmChartOptions{
			KotsAppDir:       renderDir,
			KustomizationDir: path.Join(renderDir, "overlays", "midstream"),
			RenderDir:        outDir,
		}
		err = helm.MakeHelmChart(makeHelmChartOptions)
		if err != nil {
			fmt.Printf("failed to make helm chart: %s\n", err.Error())
			ffiResult = NewFFIResult(1).WithError(err)
			return
		}

		// TODO archive the chart from outdir

		ffiResult = NewFFIResult(0)
	}()
}
