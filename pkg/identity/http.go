package identity

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
)

var (
	httpClient       *http.Client
	httpClientMu     sync.RWMutex
	kurlProxyTLSCert []byte
)

func HTTPClient(ctx context.Context, namespace string) (*http.Client, error) {
	// NOTE: it may be possible to mount both the secret and the identity spec and watch for changes

	_, err := GetConfig(ctx, namespace) // TODO
	if err != nil {
		return nil, errors.Wrap(err, "failed to get identity config")
	}

	httpClientMu.Lock()
	defer httpClientMu.Unlock()

	pemCert, err := getKurlProxyTLSCert()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kurl proxy tls cert")
	}

	if httpClient != nil && bytes.Equal(pemCert, kurlProxyTLSCert) { // TODO: identity config spec
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

	if kurlProxyTLSCert != nil {
		if !rootCAs.AppendCertsFromPEM(kurlProxyTLSCert) {
			log.Println(errors.New("no certificate blocks found in kurl tls proxy ca cert"))
		}
	}

	httpClient = cleanhttp.DefaultClient()
	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			// TODO: options from kotsv1beta1.IdentityConfigSpec
			// InsecureSkipVerify: true,
			RootCAs: rootCAs,
		},
	}

	return httpClient, nil
}

func getKurlProxyTLSCert() ([]byte, error) {
	certPath := os.Getenv("KURL_PROXY_TLS_CERT_PATH")
	if certPath == "" {
		return nil, nil
	}
	return ioutil.ReadFile(certPath)
}
