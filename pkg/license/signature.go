package license

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
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

	if wrapper.IsV1() {
		// Validate v1beta1 license using built-in ValidateLicense (MD5)
		_, err := wrapper.V1.ValidateLicense()
		if err != nil {
			return nil, errors.Wrap(err, "failed to validate v1beta1 license")
		}

		return wrapper, nil
	}

	if wrapper.IsV2() {
		// Validate v1beta2 license using built-in ValidateLicense (SHA-256)
		_, err := wrapper.V2.ValidateLicense()
		if err != nil {
			return nil, errors.Wrap(err, "failed to validate v1beta2 license")
		}

		return wrapper, nil
	}

	return nil, errors.New("license wrapper contains no license")
}

// VerifySignature validates a v1beta1 license signature using MD5.
// Deprecated: Use VerifyLicenseWrapper for multi-version support (v1beta1 and v1beta2).
// This function only supports v1beta1 licenses with MD5 signatures.
func VerifySignature(license *kotsv1beta1.License) (*kotsv1beta1.License, error) {
	outerSignature := &OuterSignature{}
	if err := json.Unmarshal(license.Spec.Signature, outerSignature); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal license outer signature")
	}

	isOldFormat := len(outerSignature.InnerSignature) == 0
	if isOldFormat {
		return verifyOldSignature(license)
	}

	innerSignature := &InnerSignature{}
	if err := json.Unmarshal(outerSignature.InnerSignature, innerSignature); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal license inner signature")
	}

	keySignature := &KeySignature{}
	if err := json.Unmarshal(innerSignature.KeySignature, keySignature); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal key signature")
	}

	globalKeyPEM, ok := PublicKeys[keySignature.GlobalKeyId]
	if !ok {
		return nil, fmt.Errorf("unknown global key")
	}

	// verify that the app public key is properly signed with a replicated private key
	if err := Verify([]byte(innerSignature.PublicKey), keySignature.Signature, globalKeyPEM); err != nil {
		return nil, errors.Wrap(err, "failed to verify key signature")
	}

	// verify that the license data is properly signed with the app private key
	if err := Verify(outerSignature.LicenseData, innerSignature.LicenseSignature, []byte(innerSignature.PublicKey)); err != nil {
		return nil, errors.Wrap(err, "failed to verify license signature")
	}

	verifiedLicense := &kotsv1beta1.License{}
	if err := json.Unmarshal(outerSignature.LicenseData, verifiedLicense); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal license data")
	}

	if err := verifyLicenseData(license, verifiedLicense); err != nil {
		return nil, LicenseDataError{message: err.Error()}
	}

	verifiedLicense.Spec.Signature = license.Spec.Signature

	return verifiedLicense, nil
}

// VerifyWithLicense validates a signature using the appropriate hash algorithm based on the license version.
// V1 licenses use MD5, V2 licenses use SHA-256. This extracts the public key from the license and verifies
// the signature using RSA-PSS.
func VerifyWithLicense(message, signature []byte, license *licensewrapper.LicenseWrapper) error {
	if license.IsEmpty() {
		return errors.New("license wrapper contains no license")
	}

	publicKeyPEM, err := GetAppPublicKey(license)
	if err != nil {
		return errors.Wrap(err, "failed to get public key from license")
	}

	pubBlock, _ := pem.Decode(publicKeyPEM)
	publicKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return errors.Wrap(err, "failed to load public key from PEM")
	}

	var opts rsa.PSSOptions
	opts.SaltLength = rsa.PSSSaltLengthAuto

	// Choose hash algorithm based on license version
	var hashFunc crypto.Hash
	if license.IsV1() {
		hashFunc = crypto.MD5
	} else {
		hashFunc = crypto.SHA256
	}

	hasher := hashFunc.New()
	hasher.Write(message)
	hashed := hasher.Sum(nil)

	err = rsa.VerifyPSS(publicKey.(*rsa.PublicKey), hashFunc, hashed, signature, &opts)
	if err != nil {
		// this ordering makes errors.Cause a little more useful
		return errors.Wrap(ErrSignatureInvalid, err.Error())
	}

	return nil
}

// Verify validates a signature using MD5 and RSA-PSS.
// Deprecated: Only supports v1beta1 MD5 signatures. Use VerifyWithLicense instead.
func Verify(message, signature, publicKeyPEM []byte) error {
	pubBlock, _ := pem.Decode(publicKeyPEM)
	publicKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return errors.Wrap(err, "failed to load public key from PEM")
	}

	var opts rsa.PSSOptions
	opts.SaltLength = rsa.PSSSaltLengthAuto

	newHash := crypto.MD5
	pssh := newHash.New()
	pssh.Write(message)
	hashed := pssh.Sum(nil)

	err = rsa.VerifyPSS(publicKey.(*rsa.PublicKey), newHash, hashed, signature, &opts)
	if err != nil {
		// this ordering makes errors.Cause a little more useful
		return errors.Wrap(ErrSignatureInvalid, err.Error())
	}

	return nil
}

// verifyLicenseData checks that the outer license fields match the inner signed fields.
// Deprecated: Only supports v1beta1 licenses. Use VerifyLicenseWrapper instead.
func verifyLicenseData(outerLicense *kotsv1beta1.License, innerLicense *kotsv1beta1.License) error {
	if outerLicense.Spec.AppSlug != innerLicense.Spec.AppSlug {
		return fmt.Errorf("\"appSlug\" field has changed to %q (license) from %q (within signature)", outerLicense.Spec.AppSlug, innerLicense.Spec.AppSlug)
	}
	if outerLicense.Spec.Endpoint != innerLicense.Spec.Endpoint {
		return fmt.Errorf("\"endpoint\" field has changed to %q (license) from %q (within signature)", outerLicense.Spec.Endpoint, innerLicense.Spec.Endpoint)
	}
	if outerLicense.Spec.CustomerName != innerLicense.Spec.CustomerName {
		return fmt.Errorf("\"CustomerName\" field has changed to %q (license) from %q (within signature)", outerLicense.Spec.CustomerName, innerLicense.Spec.CustomerName)
	}
	if outerLicense.Spec.CustomerEmail != innerLicense.Spec.CustomerEmail {
		return fmt.Errorf("\"CustomerEmail\" field has changed to %q (license) from %q (within signature)", outerLicense.Spec.CustomerEmail, innerLicense.Spec.CustomerEmail)
	}
	if outerLicense.Spec.ChannelID != innerLicense.Spec.ChannelID {
		return fmt.Errorf("\"channelID\" field has changed to %q (license) from %q (within signature)", outerLicense.Spec.ChannelID, innerLicense.Spec.ChannelID)
	}
	if outerLicense.Spec.ChannelName != innerLicense.Spec.ChannelName {
		return fmt.Errorf("\"channelName\" field has changed to %q (license) from %q (within signature)", outerLicense.Spec.ChannelName, innerLicense.Spec.ChannelName)
	}
	if outerLicense.Spec.LicenseSequence != innerLicense.Spec.LicenseSequence {
		return fmt.Errorf("\"licenseSequence\" field has changed to %q (license) from %q (within signature)", outerLicense.Spec.LicenseSequence, innerLicense.Spec.LicenseSequence)
	}
	if outerLicense.Spec.LicenseID != innerLicense.Spec.LicenseID {
		return fmt.Errorf("\"licenseID\" field has changed to %q (license) from %q (within signature)", outerLicense.Spec.LicenseID, innerLicense.Spec.LicenseID)
	}
	if outerLicense.Spec.LicenseType != innerLicense.Spec.LicenseType {
		return fmt.Errorf("\"LicenseType\" field has changed to %q (license) from %q (within signature)", outerLicense.Spec.LicenseType, innerLicense.Spec.LicenseType)
	}
	if outerLicense.Spec.IsAirgapSupported != innerLicense.Spec.IsAirgapSupported {
		return fmt.Errorf("\"IsAirgapSupported\" field has changed to %t (license) from %t (within signature)", outerLicense.Spec.IsAirgapSupported, innerLicense.Spec.IsAirgapSupported)
	}
	if outerLicense.Spec.IsGitOpsSupported != innerLicense.Spec.IsGitOpsSupported {
		return fmt.Errorf("\"IsGitOpsSupported\" field has changed to %t (license) from %t (within signature)", outerLicense.Spec.IsGitOpsSupported, innerLicense.Spec.IsGitOpsSupported)
	}
	if outerLicense.Spec.IsIdentityServiceSupported != innerLicense.Spec.IsIdentityServiceSupported {
		return fmt.Errorf("\"IsIdentityServiceSupported\" field has changed to %t (license) from %t (within signature)", outerLicense.Spec.IsIdentityServiceSupported, innerLicense.Spec.IsIdentityServiceSupported)
	}
	if outerLicense.Spec.IsGeoaxisSupported != innerLicense.Spec.IsGeoaxisSupported {
		return fmt.Errorf("\"IsGeoaxisSupported\" field has changed to %t (license) from %t (within signature)", outerLicense.Spec.IsGeoaxisSupported, innerLicense.Spec.IsGeoaxisSupported)
	}
	if outerLicense.Spec.IsSnapshotSupported != innerLicense.Spec.IsSnapshotSupported {
		return fmt.Errorf("\"IsSnapshotSupported\" field has changed to %t (license) from %t (within signature)", outerLicense.Spec.IsSnapshotSupported, innerLicense.Spec.IsSnapshotSupported)
	}
	if outerLicense.Spec.IsDisasterRecoverySupported != innerLicense.Spec.IsDisasterRecoverySupported {
		return fmt.Errorf("\"IsDisasterRecoverySupported\" field has changed to %t (license) from %t (within signature)", outerLicense.Spec.IsDisasterRecoverySupported, innerLicense.Spec.IsDisasterRecoverySupported)
	}
	if outerLicense.Spec.IsSupportBundleUploadSupported != innerLicense.Spec.IsSupportBundleUploadSupported {
		return fmt.Errorf("\"IsSupportBundleUploadSupported\" field has changed to %t (license) from %t (within signature)", outerLicense.Spec.IsSupportBundleUploadSupported, innerLicense.Spec.IsSupportBundleUploadSupported)
	}
	if outerLicense.Spec.IsSemverRequired != innerLicense.Spec.IsSemverRequired {
		return fmt.Errorf("\"IsSemverRequired\" field has changed to %t (license) from %t (within signature)", outerLicense.Spec.IsSemverRequired, innerLicense.Spec.IsSemverRequired)
	}

	// Check entitlements
	if len(outerLicense.Spec.Entitlements) != len(innerLicense.Spec.Entitlements) {
		return fmt.Errorf("\"entitlements\" field length has changed to %d (license) from %d (within signature)", len(outerLicense.Spec.Entitlements), len(innerLicense.Spec.Entitlements))
	}
	for k, outerEntitlement := range outerLicense.Spec.Entitlements {
		innerEntitlement, ok := innerLicense.Spec.Entitlements[k]
		if !ok {
			return fmt.Errorf("entitlement %q not found in the inner license", k)
		}
		if outerEntitlement.Value.Value() != innerEntitlement.Value.Value() {
			return fmt.Errorf("entitlement %q value has changed to %q (license) from %q (within signature)", k, outerEntitlement.Value.Value(), innerEntitlement.Value.Value())
		}
		if outerEntitlement.Title != innerEntitlement.Title {
			return fmt.Errorf("entitlement %q title has changed to %q (license) from %q (within signature)", k, outerEntitlement.Title, innerEntitlement.Title)
		}
		if outerEntitlement.Description != innerEntitlement.Description {
			return fmt.Errorf("entitlement %q description has changed to %q (license) from %q (within signature)", k, outerEntitlement.Description, innerEntitlement.Description)
		}
		if outerEntitlement.IsHidden != innerEntitlement.IsHidden {
			return fmt.Errorf("entitlement %q hidden has changed to %t (license) from %t (within signature)", k, outerEntitlement.IsHidden, innerEntitlement.IsHidden)
		}
		if outerEntitlement.ValueType != innerEntitlement.ValueType {
			return fmt.Errorf("entitlement %q value type has changed to %q (license) from %q (within signature)", k, outerEntitlement.ValueType, innerEntitlement.ValueType)
		}
	}

	return nil
}

// verifyOldSignature validates licenses with the old signature format.
// Deprecated: Only supports v1beta1 legacy format. Use VerifyLicenseWrapper instead.
func verifyOldSignature(license *kotsv1beta1.License) (*kotsv1beta1.License, error) {
	signature := &InnerSignature{}
	if err := json.Unmarshal(license.Spec.Signature, signature); err != nil {
		// old licenses's signature is a single space character
		if len(license.Spec.Signature) == 0 || len(license.Spec.Signature) == 1 {
			return nil, ErrSignatureMissing
		}
		return nil, errors.Wrap(err, "failed to unmarshal license signature")
	}

	keySignature := &KeySignature{}
	if err := json.Unmarshal(signature.KeySignature, keySignature); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal key signature")
	}

	globalKeyPEM, ok := PublicKeys[keySignature.GlobalKeyId]
	if !ok {
		return nil, fmt.Errorf("unknown global key")
	}

	if err := Verify([]byte(signature.PublicKey), keySignature.Signature, globalKeyPEM); err != nil {
		return nil, errors.Wrap(err, "failed to verify key signature")
	}

	licenseMessage, err := getMessageFromLicense(license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert license to message")
	}

	if err := Verify(licenseMessage, signature.LicenseSignature, []byte(signature.PublicKey)); err != nil {
		return nil, errors.Wrap(err, "failed to verify license signature")
	}

	return license, nil
}

func getMessageFromLicense(license *kotsv1beta1.License) ([]byte, error) {
	// JSON marshaller will sort map keys automatically.
	fields := map[string]string{
		"apiVersion":             license.APIVersion,
		"kind":                   license.Kind,
		"metadata.name":          license.GetObjectMeta().GetName(),
		"spec.licenseID":         license.Spec.LicenseID,
		"spec.appSlug":           license.Spec.AppSlug,
		"spec.channelName":       license.Spec.ChannelName,
		"spec.endpoint":          license.Spec.Endpoint,
		"spec.isAirgapSupported": fmt.Sprintf("%t", license.Spec.IsAirgapSupported),
	}

	if license.Spec.LicenseSequence > 0 {
		fields["spec.licenseSequence"] = fmt.Sprintf("%d", license.Spec.LicenseSequence)
	}

	for k, v := range license.Spec.Entitlements {
		key := fmt.Sprintf("spec.entitlements.%s", k)
		val := map[string]string{
			"title":       v.Title,
			"description": v.Description,
			"value":       fmt.Sprintf("%v", v.Value.Value()),
		}
		valStr, err := json.Marshal(val)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal entitlement value: %s", k)
		}
		fields[key] = string(valStr)
	}

	message, err := json.Marshal(fields)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal message JSON")
	}

	return message, err
}

func GetAppPublicKey(license *licensewrapper.LicenseWrapper) ([]byte, error) {
	signature := license.GetSignature()
	// old licenses's signature is a single space character
	if len(signature) == 0 || len(signature) == 1 {
		return nil, ErrSignatureMissing
	}

	innerSignature := &InnerSignature{}

	outerSignature := &OuterSignature{}
	if err := json.Unmarshal(signature, outerSignature); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal license outer signature")
	}

	isOldFormat := len(outerSignature.InnerSignature) == 0
	if isOldFormat {
		if err := json.Unmarshal(signature, innerSignature); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal license signature")
		}
	} else {
		if err := json.Unmarshal(outerSignature.InnerSignature, innerSignature); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal license inner signature")
		}
	}

	return []byte(innerSignature.PublicKey), nil
}
