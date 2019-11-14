package main

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

func writeLicenseFileFromLicenseData(licenseData string) (string, error) {
	licenseFile, err := ioutil.TempFile("", "kots")
	if err != nil {
		return "", errors.Wrapf(err, "failed to create temp file")
	}

	if err := ioutil.WriteFile(licenseFile.Name(), []byte(licenseData), 0644); err != nil {
		os.Remove(licenseFile.Name())
		return "", errors.Wrapf(err, "failed to write license to temp file")
	}

	return licenseFile.Name(), nil
}

func loadLicenseFromPath(expectedLicenseFile string) (*kotsv1beta1.License, error) {
	_, err := os.Stat(expectedLicenseFile)
	if err != nil {
		return nil, errors.New("find license file in archive")
	}
	licenseData, err := ioutil.ReadFile(expectedLicenseFile)
	if err != nil {
		return nil, errors.Wrap(err, "read license file")
	}

	return loadLicense(string(licenseData))
}

func loadLicense(licenseData string) (*kotsv1beta1.License, error) {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(licenseData), nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "decode license data")
	}

	return obj.(*kotsv1beta1.License), nil
}
