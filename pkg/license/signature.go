package license

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper"
	"github.com/replicatedhq/kotskinds/pkg/licensewrapper/types"
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
// Behavior:
// - Cryptographic signature failures (invalid/tampered signature): Returns an error
// - Data validation errors (field mismatch between outer license and signed inner license):
//   Logs a warning but returns success. This handles cases where Replicated SaaS adds fields
//   to the signature that KOTS doesn't know about or defaults differently.
//
// Note: This function validates the license signature only. Entitlement signature validation
// is handled separately where needed, matching the behavior of the deprecated VerifySignature function.
func VerifyLicenseWrapper(wrapper *licensewrapper.LicenseWrapper) (*licensewrapper.LicenseWrapper, error) {
	if wrapper.IsEmpty() {
		return nil, errors.New("license wrapper contains no license")
	}

	err := wrapper.VerifySignature()
	if err != nil {
		if types.IsLicenseDataValidationError(err) {
			// Non-fatal: Field mismatch between outer license (YAML) and signed inner license.
			// The cryptographic signature is valid, but some field values differ from what was signed.
			// This can happen when Replicated SaaS adds new fields to signatures that KOTS doesn't
			// know about. Log a warning but allow the license to be used with the signed values.
			logger.Warnf("License data validation warning: %s", err.Error())
		} else {
			// Fatal: Cryptographic signature verification failed (invalid or tampered signature)
			return nil, err
		}
	}

	return wrapper, nil
}
