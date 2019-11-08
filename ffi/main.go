package main

import "C"

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/upstream"
	"k8s.io/client-go/kubernetes/scheme"
)

//export UpdateCheck
func UpdateCheck(socket, fromArchivePath string) {
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

		tmpRoot, err := ioutil.TempDir("", "kots")
		if err != nil {
			fmt.Printf("failed to create temp path: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}
		defer os.RemoveAll(tmpRoot)

		// extract the current archive to this root
		tarGz := archiver.TarGz{
			Tar: &archiver.Tar{
				ImplicitTopLevelFolder: false,
			},
		}
		if err := tarGz.Unarchive(fromArchivePath, tmpRoot); err != nil {
			fmt.Printf("failed to unarchive: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		beforeCursor, err := readCursorFromPath(tmpRoot)
		if err != nil {
			fmt.Printf("failed to read cursor file: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		expectedLicenseFile := path.Join(tmpRoot, "upstream", "userdata", "license.yaml")
		_, err = os.Stat(expectedLicenseFile)
		if err != nil {
			fmt.Printf("failed to find license file in archive\n")
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}
		licenseData, err := ioutil.ReadFile(expectedLicenseFile)
		if err != nil {
			fmt.Printf("failed to read license file: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		kotsscheme.AddToScheme(scheme.Scheme)
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(licenseData), nil, nil)
		if err != nil {
			fmt.Printf("failed to decode license data: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}
		license := obj.(*kotsv1beta1.License)

		pullOptions := pull.PullOptions{
			LicenseFile:         expectedLicenseFile,
			RootDir:             tmpRoot,
			ExcludeKotsKinds:    true,
			ExcludeAdminConsole: true,
			CreateAppDir:        false,
		}

		if _, err := pull.Pull(fmt.Sprintf("replicated://%s", license.Spec.AppSlug), pullOptions); err != nil {
			fmt.Printf("failed to pull upstream: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		afterCursor, err := readCursorFromPath(tmpRoot)
		if err != nil {
			fmt.Printf("failed to read cursor file after update: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		fmt.Printf("Result of checking for updates for %s: Before: %s, After %s\n", license.Spec.AppSlug, beforeCursor, afterCursor)

		isUpdateAvailable := string(beforeCursor) != string(afterCursor)
		if !isUpdateAvailable {
			ffiResult = NewFFIResult(0)
			return
		}

		paths := []string{
			path.Join(tmpRoot, "upstream"),
			path.Join(tmpRoot, "base"),
			path.Join(tmpRoot, "overlays"),
		}

		err = os.Remove(fromArchivePath)
		if err != nil {
			fmt.Printf("failed to delete archive to replace: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		if err := tarGz.Archive(paths, fromArchivePath); err != nil {
			fmt.Printf("failed to write archive: %s\n", err.Error())
			ffiResult = NewFFIResult(-1).WithError(err)
			return
		}

		ffiResult = NewFFIResult(1)
	}()
}

//export GetLatestLicense
func GetLatestLicense(licenseData string) *C.char {
	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(licenseData), nil, nil)
	if err != nil {
		fmt.Printf("failed to decode license data: %s\n", err.Error())
		return C.CString("")
	}
	license := obj.(*kotsv1beta1.License)

	u, err := url.Parse(license.Spec.Endpoint)
	if err != nil {
		fmt.Printf("failed to parse endpoint from license: %s\n", err.Error())
		return C.CString("")
	}

	hostname := u.Hostname()
	if u.Port() != "" {
		hostname = fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	}

	url := fmt.Sprintf("%s://%s/release/%s/license", u.Scheme, hostname, license.Spec.AppSlug)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("failed to call newrequest: %s\n", err.Error())
		return C.CString("")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID)))))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to execute get request: %s\n", err.Error())
		return C.CString("")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Printf("unexpected result from get request: %d\n", resp.StatusCode)
		return C.CString("")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("failed to load response")
		return C.CString("")
	}

	obj, _, err = decode(body, nil, nil)
	if err != nil {
		fmt.Printf("failed to decode latest license data: %s\n", err.Error())
		return C.CString("")
	}
	latestLicense := obj.(*kotsv1beta1.License)

	marshalledLicense := upstream.MustMarshalLicense(latestLicense)

	return C.CString(string(marshalledLicense))
}

//export VerifyAirgapLicense
func VerifyAirgapLicense(licenseData string) *C.char {
	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(licenseData), nil, nil)
	if err != nil {
		fmt.Printf("failed to decode license data: %s\n", err.Error())
		return C.CString("")
	}
	license := obj.(*kotsv1beta1.License)

	if err := pull.VerifySignature(license); err != nil {
		fmt.Printf("failed to verify airgap license signature: %s\n", err.Error())
		return C.CString("")
	}

	return C.CString("verified")
}

func readCursorFromPath(rootPath string) (string, error) {
	installationFilePath := path.Join(rootPath, "upstream", "userdata", "installation.yaml")
	_, err := os.Stat(installationFilePath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", errors.Wrap(err, "failed to open file")
	}

	installationData, err := ioutil.ReadFile(installationFilePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read update installation file")
	}

	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(installationData), nil, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to devode installation data")
	}

	installation := obj.(*kotsv1beta1.Installation)
	return installation.Spec.UpdateCursor, nil
}

func main() {}
