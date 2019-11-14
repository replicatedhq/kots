package pull

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
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
	LicenseData    string `json:"licenseData"`
	InnerSignature string `json:"innerSignature"`
}

type KeySignature struct {
	Signature   []byte `json:"signature"`
	GlobalKeyId string `json:"globalKeyId"`
}

func VerifySignature(license *kotsv1beta1.License) error {
	decodedOuterSignature, err := base64.StdEncoding.DecodeString(license.Spec.Signature)
	if err != nil {
		return errors.New("failed to decode license signature")
	}

	outerSignature := &OuterSignature{}
	if err := json.Unmarshal(decodedOuterSignature, outerSignature); err != nil {
		return errors.Wrap(err, "failed to unmarshal license outer signature")
	}

	decodedInnerSignature, err := base64.StdEncoding.DecodeString(outerSignature.InnerSignature)
	if err != nil {
		return errors.New("failed to decode license signature")
	}

	innerSignature := &InnerSignature{}
	if err := json.Unmarshal(decodedInnerSignature, innerSignature); err != nil {
		// old licenses's signature is a single space character
		if len(decodedInnerSignature) == 0 || len(decodedInnerSignature) == 1 {
			return ErrSignatureMissing
		}
		return errors.Wrap(err, "failed to unmarshal license inner signature")
	}

	keySignature := &KeySignature{}
	if err := json.Unmarshal(innerSignature.KeySignature, keySignature); err != nil {
		return errors.Wrap(err, "failed to unmarshal key signature")
	}

	globalKeyPEM, ok := publicKeys[keySignature.GlobalKeyId]
	if !ok {
		return errors.New("unknown global key")
	}

	if err := verify([]byte(innerSignature.PublicKey), keySignature.Signature, globalKeyPEM); err != nil {
		return errors.Wrap(err, "failed to verify key signature")
	}

	licenseMessage, err := base64.StdEncoding.DecodeString(outerSignature.LicenseData)
	if err != nil {
		return errors.New("failed to decode license data from outer signature")
	}

	if err := verify(licenseMessage, innerSignature.LicenseSignature, []byte(innerSignature.PublicKey)); err != nil {
		return errors.Wrap(err, "failed to verify license signature")
	}

	return nil
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
