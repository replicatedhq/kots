package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/cloudflare/cfssl/initca"
	"github.com/cloudflare/cfssl/signer"
	"github.com/cloudflare/cfssl/signer/local"
	"github.com/pkg/errors"
)

type CertType struct {
	Cert string
	Key  string
}

type CAType struct {
	Cert string
	Key  string
}

func makeKeyRequest(certKind string) (csr.BasicKeyRequest, error) {
	kindParts := strings.Split(certKind, "-")
	if len(kindParts) == 2 {
		if kindParts[0] == "rsa" {
			rsaBits, err := strconv.ParseInt(kindParts[1], 10, 32)
			if err != nil {
				return csr.BasicKeyRequest{}, errors.Wrapf(err, "unable to parse kind %s", certKind)
			}
			return csr.BasicKeyRequest{A: "rsa", S: int(rsaBits)}, nil
		}
	} else if len(kindParts) == 1 {
		switch certKind {
		case "":
			// use 2048 bit rsa if no key type is specified
			return csr.BasicKeyRequest{A: "rsa", S: 2048}, nil
		// elliptic curve keys
		case "P256":
			return csr.BasicKeyRequest{A: "ecdsa", S: 256}, nil
		case "P384":
			return csr.BasicKeyRequest{A: "ecdsa", S: 384}, nil
		case "P521":
			return csr.BasicKeyRequest{A: "ecdsa", S: 521}, nil
		default:
		}
	}
	return csr.BasicKeyRequest{}, fmt.Errorf("unable to parse kind %s", certKind)

}

func MakeCert(host []string, certKind, CACert, CAKey string) (CertType, error) {
	parsedCaCert, err := helpers.ParseCertificatePEM([]byte(CACert))
	if err != nil {
		return CertType{}, errors.Wrap(err, "parse cert pem")
	}
	parsedCaKey, err := helpers.ParsePrivateKeyPEM([]byte(CAKey))
	if err != nil {
		return CertType{}, errors.Wrap(err, "parse key pem")
	}

	keyRequest, err := makeKeyRequest(certKind)
	if err != nil {
		return CertType{}, errors.Wrap(err, "parse kind")
	}

	req := csr.CertificateRequest{
		KeyRequest: &keyRequest,
		Hosts:      host,
	}
	certReq, key, err := csr.ParseRequest(&req)
	if err != nil {
		return CertType{}, errors.Wrap(err, "parse csr")
	}

	localSigner, err := local.NewSigner(parsedCaKey, parsedCaCert, signer.DefaultSigAlgo(parsedCaKey), nil)
	if err != nil {
		return CertType{}, errors.Wrap(err, "create signer")
	}

	signedCert, err := localSigner.Sign(signer.SignRequest{Hosts: host, Request: string(certReq), NotBefore: time.Now()})
	if err != nil {
		return CertType{}, errors.Wrap(err, "sign request")
	}

	return CertType{Cert: string(signedCert), Key: string(key)}, nil
}

func MakeCA(caKind string) (CAType, error) {
	keyRequest, err := makeKeyRequest(caKind)
	if err != nil {
		return CAType{}, errors.Wrap(err, "parse kind")
	}

	req := csr.CertificateRequest{
		KeyRequest: &keyRequest,
		CN:         "gatekeeper_ca",
		Hosts: []string{
			"gatekeeper_ca",
		},
		CA: &csr.CAConfig{
			Expiry: "43800h", // 5 years
		},
	}

	cert, _, key, err := initca.New(&req)
	if err != nil {
		return CAType{}, errors.Wrap(err, "initca")
	}

	return CAType{Cert: string(cert), Key: string(key)}, nil
}
