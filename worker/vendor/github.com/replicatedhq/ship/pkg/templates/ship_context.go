package templates

import (
	"strings"
	"text/template"

	"github.com/go-kit/kit/log"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util"
)

var amazonEKSPaths map[string]string
var googleGKEPaths map[string]string
var azureAKSPaths map[string]string

// ShipContext is the context for builder functions that depend on what assets have been created.
type ShipContext struct {
	Logger  log.Logger
	Manager state.Manager
}

func (bb *BuilderBuilder) NewShipContext() (*ShipContext, error) {
	shipCtx := &ShipContext{
		Logger:  bb.Logger,
		Manager: bb.Manager,
	}

	return shipCtx, nil
}

// FuncMap represents the available functions in the ShipCtx.
func (ctx ShipContext) FuncMap() template.FuncMap {
	return template.FuncMap{
		"AmazonEKS": ctx.amazonEKS,
		"GoogleGKE": ctx.googleGKE,
		"AzureAKS":  ctx.azureAKS,
		"GetCaKey":  ctx.makeCaKey,
		"GetCaCert": ctx.getCaCert,
		"GetKey":    ctx.makeCertKey,
		"GetCert":   ctx.getCert,
	}
}

// amazonEKS returns the path within the InstallerPrefixPath that the kubeconfig for the named cluster can be found at
func (ctx ShipContext) amazonEKS(name string) string {
	return amazonEKSPaths[name]
}

// AddAmazonEKSPath adds a kubeconfig path to the cache
func AddAmazonEKSPath(name string, path string) {
	if amazonEKSPaths == nil {
		amazonEKSPaths = make(map[string]string)
	}
	amazonEKSPaths[name] = path
}

// googleGKE returns the path within the InstallerPrefixPath that the kubeconfig for the named cluster can be found at
func (ctx ShipContext) googleGKE(name string) string {
	return googleGKEPaths[name]
}

// AddGoogleGKEPath adds a kubeconfig path to the cache
func AddGoogleGKEPath(name string, path string) {
	if googleGKEPaths == nil {
		googleGKEPaths = make(map[string]string)
	}
	googleGKEPaths[name] = path
}

// azureAKS returns the path within the InstallerPrefixPath that the kubeconfig for the named cluster can be found at
func (ctx ShipContext) azureAKS(name string) string {
	return azureAKSPaths[name]
}

// AddAzureAKSPath adds a kubeconfig path to the cache
func AddAzureAKSPath(name string, path string) {
	if azureAKSPaths == nil {
		azureAKSPaths = make(map[string]string)
	}
	azureAKSPaths[name] = path
}

// Certificate generation functions

// certs can be rsa or ecdsa, with a default of rsa-2048. Acceptable entries are rsa-n (where n is the number of bits),
// P256, P384, or P521.
func (ctx ShipContext) makeCaKey(caName string, caType string) string {
	// check if the CA exists - if it does, use the existing one
	// if it does not, make a new one
	currentState, err := ctx.Manager.CachedState()
	if err != nil {
		ctx.Logger.Log("method", "makeCaKey", "action", "tryLoad", "error", err)
		return ""
	}
	CAs := currentState.CurrentCAs()
	if CAs != nil {
		if ca, ok := CAs[caName]; ok {
			return ca.Key
		}
	}

	newCA, err := util.MakeCA(caType)
	if err != nil {
		ctx.Logger.Log("method", "makeCaKey", "action", "makeCA", "error", err)
		return ""
	}

	err = ctx.Manager.AddCA(caName, newCA)
	if err != nil {
		ctx.Logger.Log("method", "makeCaKey", "action", "saveCA", "error", err)
		return ""
	}

	return newCA.Key
}

func (ctx ShipContext) getCaCert(caName string) string {
	currentState, err := ctx.Manager.CachedState()
	if err != nil {
		ctx.Logger.Log("method", "getCaCert", "action", "tryLoad", "error", err)
		return ""
	}
	CAs := currentState.CurrentCAs()
	if CAs != nil {
		if ca, ok := CAs[caName]; ok {
			return ca.Cert
		}
	}

	ctx.Logger.Log("method", "getCaCert", "error", "certNotPresent")
	return ""
}

// certs can be rsa or ecdsa, with a default of rsa-2048. Acceptable entries are rsa-n (where n is the number of bits),
// P256, P384, or P521.
func (ctx ShipContext) makeCertKey(certName string, caName string, hosts string, certType string) string {
	// try to find existing cert to use
	currentState, err := ctx.Manager.CachedState()
	if err != nil {
		ctx.Logger.Log("method", "makeCertKey", "action", "tryLoad", "error", err)
		return ""
	}
	certs := currentState.CurrentCerts()
	if certs != nil {
		if cert, ok := certs[certName]; ok {
			return cert.Key
		}
	}

	// cert does not yet exist - get CA and make a cert with it
	caKey := ctx.makeCaKey(caName, certType)
	if caKey == "" {
		ctx.Logger.Log("method", "makeCertKey", "error", "caKeyNotPresent")
		return ""
	}
	caCert := ctx.getCaCert(caName)
	if caCert == "" {
		ctx.Logger.Log("method", "makeCertKey", "error", "caCertNotPresent")
		return ""
	}

	hostList := strings.Split(hosts, ",")

	newCert, err := util.MakeCert(hostList, certType, caCert, caKey)
	if err != nil {
		ctx.Logger.Log("method", "makeCertKey", "action", "makeCert", "error", err)
		return ""
	}

	err = ctx.Manager.AddCert(certName, newCert)
	if err != nil {
		ctx.Logger.Log("method", "makeCertKey", "action", "saveCA", "error", err)
		return ""
	}

	return newCert.Key
}

func (ctx ShipContext) getCert(certName string) string {
	currentState, err := ctx.Manager.CachedState()
	if err != nil {
		ctx.Logger.Log("method", "getCert", "action", "tryLoad", "error", err)
		return ""
	}
	Certs := currentState.CurrentCerts()
	if Certs != nil {
		if cert, ok := Certs[certName]; ok {
			return cert.Cert
		}
	}

	ctx.Logger.Log("method", "getCert", "error", "certNotPresent")
	return ""
}
