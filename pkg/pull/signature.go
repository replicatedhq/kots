package pull

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

var (
	ErrSignatureInvalid = errors.New("license signature is invalid")
	ErrSignatureMissing = errors.New("license signature is missing")
)

type Signature struct {
	LicenseSignature []byte `json:"licenseSignature"`
	PublicKey        string `json:"publicKey"`
	KeySignature     []byte `json:"keySignature"`
}

type KeySignature struct {
	Signature   []byte `json:"signature"`
	GlobalKeyId string `json:"globalKeyId"`
}

func verifySignature(license *kotsv1beta1.License) error {
	signature := &Signature{}
	if err := json.Unmarshal(license.Spec.Signature, signature); err != nil {
		// old licenses's signature is a single space character
		if len(license.Spec.Signature) == 0 || len(license.Spec.Signature) == 1 {
			return ErrSignatureMissing
		}
		return errors.Wrap(err, "failed to unmarshal license signature")
	}

	keySignature := &KeySignature{}
	if err := json.Unmarshal(signature.KeySignature, keySignature); err != nil {
		return errors.Wrap(err, "failed to unmarshal key signature")
	}

	globalKeyPEM, ok := publicKeys[keySignature.GlobalKeyId]
	if !ok {
		return errors.New("unknown global key")
	}

	if err := verify([]byte(signature.PublicKey), keySignature.Signature, globalKeyPEM); err != nil {
		return errors.Wrap(err, "failed to verify key signature")
	}

	licenseMessage, err := getMessageFromLicense(license)
	if err != nil {
		return errors.Wrap(err, "failed to convert license to message")
	}

	if err := verify(licenseMessage, signature.LicenseSignature, []byte(signature.PublicKey)); err != nil {
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
