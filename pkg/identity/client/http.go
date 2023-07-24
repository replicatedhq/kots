package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

var (
	httpClient                              *http.Client
	httpClientMu                            sync.RWMutex
	kurlProxyTLSCert                        []byte
	identityConfigSpecCACert                string
	identityConfigSpecInsecureSkipTLSVerify bool
)

func HTTPClient(ctx context.Context, namespace string, identityConfig kotsv1beta1.IdentityConfig) (*http.Client, error) {
	// NOTE: it may be possible to mount both the secret and the identity spec and watch for changes

	httpClientMu.Lock()
	defer httpClientMu.Unlock()

	pemCert, err := getKurlProxyTLSCert()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kurl proxy tls cert")
	}

	if httpClient != nil && !hasTLSConfigChanged(pemCert, identityConfig.Spec) {
		return httpClient, nil
	}

	// Get the SystemCertPool, continue with an empty pool on error
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get system cert pool")
	}
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	kurlProxyTLSCert = pemCert
	identityConfigSpecCACert = identityConfig.Spec.CACertPemBase64
	identityConfigSpecInsecureSkipTLSVerify = identityConfig.Spec.InsecureSkipTLSVerify

	if kurlProxyTLSCert != nil {
		if !rootCAs.AppendCertsFromPEM(kurlProxyTLSCert) {
			// TODO: how can I log here?
			log.Println(errors.New("no certificate blocks found in kurl tls proxy ca certificate"))
		}
	}

	if identityConfig.Spec.CACertPemBase64 != "" {
		caCert, err := base64.StdEncoding.DecodeString(identityConfig.Spec.CACertPemBase64)
		if err != nil {
			log.Println(errors.Wrap(err, "failed to base64 decode provided ca certificate"))
		} else if caCert != nil {
			if !rootCAs.AppendCertsFromPEM(caCert) {
				log.Println(errors.New("no certificate blocks found in provided ca certificate"))
			}
		}
	}

	transport := cleanhttp.DefaultTransport()
	transport.TLSClientConfig = &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: identityConfig.Spec.InsecureSkipTLSVerify,
	}

	httpClient = &http.Client{
		Transport: transport,
	}

	return httpClient, nil
}

func hasTLSConfigChanged(pemCert []byte, identityConfigSpec kotsv1beta1.IdentityConfigSpec) bool {
	return !bytes.Equal(kurlProxyTLSCert, pemCert) ||
		identityConfigSpecCACert != identityConfigSpec.CACertPemBase64 ||
		identityConfigSpecInsecureSkipTLSVerify != identityConfigSpec.InsecureSkipTLSVerify
}

func getKurlProxyTLSCert() ([]byte, error) {
	certPath := os.Getenv("KURL_PROXY_TLS_CERT_PATH")
	if certPath == "" {
		return nil, nil
	}
	return ioutil.ReadFile(certPath)
}
