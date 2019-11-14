package pull

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

var (
	ErrSignatureInvalid = errors.New("license signature is invalid")
	ErrSignatureMissing = errors.New("license signature is missing")
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

func VerifySignature(license *kotsv1beta1.License) (*kotsv1beta1.License, error) {
	outerSignature := &OuterSignature{}
	if err := json.Unmarshal(license.Spec.Signature, outerSignature); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal license outer signature")
	}

	innerSignature := &InnerSignature{}
	if err := json.Unmarshal(outerSignature.InnerSignature, innerSignature); err != nil {
		// old licenses's signature is a single space character
		if len(outerSignature.InnerSignature) == 0 || len(outerSignature.InnerSignature) == 1 {
			return nil, ErrSignatureMissing
		}
		return nil, errors.Wrap(err, "failed to unmarshal license inner signature")
	}

	keySignature := &KeySignature{}
	if err := json.Unmarshal(innerSignature.KeySignature, keySignature); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal key signature")
	}

	globalKeyPEM, ok := publicKeys[keySignature.GlobalKeyId]
	if !ok {
		return nil, errors.New("unknown global key")
	}

	if err := verify([]byte(innerSignature.PublicKey), keySignature.Signature, globalKeyPEM); err != nil {
		return nil, errors.Wrap(err, "failed to verify key signature")
	}

	if err := verify(outerSignature.LicenseData, innerSignature.LicenseSignature, []byte(innerSignature.PublicKey)); err != nil {
		return nil, errors.Wrap(err, "failed to verify license signature")
	}

	verifiedLicense := &kotsv1beta1.License{}
	if err := json.Unmarshal(outerSignature.LicenseData, verifiedLicense); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal license data")
	}

	if err := verifyLicenseData(license, verifiedLicense); err != nil {
		return nil, errors.Wrap(err, "failed to verify license data")
	}

	return verifiedLicense, nil
}

func verify(message, signature, publicKeyPEM []byte) error {
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

func verifyLicenseData(outerLicense *kotsv1beta1.License, innerLicense *kotsv1beta1.License) error {
	if outerLicense.Spec.AppSlug != innerLicense.Spec.AppSlug {
		return errors.New("\"appSlug\" field has changed")
	}
	if outerLicense.Spec.Endpoint != innerLicense.Spec.Endpoint {
		return errors.New("\"endpoint\" field has changed")
	}
	if outerLicense.Spec.ChannelName != innerLicense.Spec.ChannelName {
		return errors.New("\"channelName\" field has changed")
	}
	if outerLicense.Spec.LicenseSequence != innerLicense.Spec.LicenseSequence {
		return errors.New("\"licenseSequence\" field has changed")
	}
	if outerLicense.Spec.LicenseID != innerLicense.Spec.LicenseID {
		return errors.New("\"licenseID\" field has changed")
	}
	if outerLicense.Spec.IsAirgapSupported != innerLicense.Spec.IsAirgapSupported {
		return errors.New("\"isAirgapSupported\" field has changed")
	}

	// Check entitlements
	if len(outerLicense.Spec.Entitlements) != len(innerLicense.Spec.Entitlements) {
		return errors.New("\"entitlements\" field has changed")
	}
	for k, outerEntitlement := range outerLicense.Spec.Entitlements {
		innerEntitlement, ok := innerLicense.Spec.Entitlements[k]
		if !ok {
			return errors.New("entitlement not found in the inner license")
		}
		if outerEntitlement.Value.Value() != innerEntitlement.Value.Value() {
			return errors.New("one or more of the entitlements values have changed")
		}
		if outerEntitlement.Title != innerEntitlement.Title {
			return errors.New("one or more of the entitlements titles have changed")
		}
		if outerEntitlement.Description != innerEntitlement.Description {
			return errors.New("one or more of the entitlements descriptions have changed")
		}
		if outerEntitlement.IsHidden != innerEntitlement.IsHidden {
			return errors.New("one or more of the hidden entitlements have changed")
		}
	}

	return nil
}
