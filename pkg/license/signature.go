package license

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
)

var (
	ErrSignatureInvalid = fmt.Errorf("signature is invalid")
	ErrSignatureMissing = fmt.Errorf("signature is missing")
)

type InnerSignature struct {
	LicenseSignature []byte `json:"licenseSignature"`
	PublicKey        string `json:"publicKey"`
	KeySignature     []byte `json:"keySignature"`
}

type OuterSignature struct {
	LicenseData    []byte `json:"licenseData"`
	InnerSignature []byte `json:"innerSignature"`
}

type KeySignature struct {
	Signature   []byte `json:"signature"`
	GlobalKeyId string `json:"globalKeyId"`
}

type LicenseDataError struct {
	message string
}

func (e LicenseDataError) Error() string {
	return e.message
}

// VerifyLicenseWrapper validates a license wrapper by delegating to the appropriate
// version-specific validation method. Returns the same wrapper if validation succeeds.
// This function supports both v1beta1 (MD5) and v1beta2 (SHA-256) licenses.
//
// Note: This function validates the license signature only. Entitlement signature validation
// is handled separately where needed, matching the behavior of the deprecated VerifySignature function.
func VerifyLicenseWrapper(wrapper *licensewrapper.LicenseWrapper) (*licensewrapper.LicenseWrapper, error) {
	if wrapper.IsEmpty() {
		return nil, errors.New("license wrapper contains no license")
	}

  return wrapper, wrapper.VerifySignature()
}

func VerifyWithLicense(message, signature []byte, license *licensewrapper.LicenseWrapper) error {
	if license.IsEmpty() {
		return errors.New("license wrapper contains no license")
	}

  return license.VerifySignature() 
}
