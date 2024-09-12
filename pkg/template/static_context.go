package template

/*
  This was taken from https://github.com/replicatedhq/replicated/blob/main/templates/context.go
*/

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	sprig "github.com/Masterminds/sprig/v3"
	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/util"
	analyze "github.com/replicatedhq/troubleshoot/pkg/analyze"
	"gopkg.in/yaml.v3"
	helmengine "helm.sh/helm/v3/pkg/engine"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	certUtil "k8s.io/client-go/util/cert"
)

type Ctx interface {
	FuncMap() template.FuncMap
}

type StaticCtx struct {
	// a new clientset will be initialized if nil
	clientset kubernetes.Interface
}

type TLSPair struct {
	Cert string
	Key  string
	Cn   string
}

var tlsMap = map[string]TLSPair{}
var caMap = map[string]TLSPair{}

func (ctx StaticCtx) FuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()

	funcMap["Now"] = ctx.now
	funcMap["NowFmt"] = ctx.nowFormat
	funcMap["ToLower"] = strings.ToLower
	funcMap["ToUpper"] = strings.ToUpper
	funcMap["TrimSpace"] = strings.TrimSpace
	funcMap["Trim"] = ctx.trim
	funcMap["UrlEncode"] = url.QueryEscape
	funcMap["Base64Encode"] = ctx.base64Encode
	funcMap["Base64Decode"] = ctx.base64Decode
	funcMap["Split"] = strings.Split
	funcMap["RandomBytes"] = ctx.RandomBytes
	funcMap["RandomString"] = ctx.RandomString
	funcMap["Add"] = ctx.add
	funcMap["Sub"] = ctx.sub
	funcMap["Mult"] = ctx.mult
	funcMap["Div"] = ctx.div
	funcMap["ParseBool"] = ctx.parseBool
	funcMap["ParseFloat"] = ctx.parseFloat
	funcMap["ParseInt"] = ctx.parseInt
	funcMap["ParseUint"] = ctx.parseUint
	funcMap["HumanSize"] = ctx.humanSize
	funcMap["KubeSeal"] = ctx.kubeSeal
	funcMap["Namespace"] = ctx.namespace

	funcMap["TLSCert"] = ctx.tlsCert
	funcMap["TLSKey"] = ctx.tlsKey

	funcMap["TLSCACert"] = ctx.tlsCaCert
	funcMap["TLSCertFromCA"] = ctx.tlsCertFromCa
	funcMap["TLSKeyFromCA"] = ctx.tlsKeyFromCa

	funcMap["KotsVersion"] = ctx.kotsVersion
	funcMap["IsKurl"] = ctx.isKurl
	funcMap["Distribution"] = ctx.distribution
	funcMap["NodeCount"] = ctx.nodeCount

	funcMap["HTTPSProxy"] = ctx.httpsProxy
	funcMap["HTTPProxy"] = ctx.httpProxy
	funcMap["NoProxy"] = ctx.noProxy

	funcMap["YamlEscape"] = ctx.yamlEscape

	funcMap["KubernetesVersion"] = ctx.kubernetesVersion
	funcMap["KubernetesMajorVersion"] = ctx.kubernetesMajorVersion
	funcMap["KubernetesMinorVersion"] = ctx.kubernetesMinorVersion

	funcMap["Lookup"] = ctx.lookup

	funcMap["PrivateCACert"] = ctx.privateCACert
	funcMap["PrivateCACertNamespace"] = ctx.privateCACertNamespace

	return funcMap
}

func (ctx StaticCtx) now() string {
	return ctx.nowFormat("")
}

func (ctx StaticCtx) nowFormat(format string) string {
	if format == "" {
		format = time.RFC3339
	}
	return time.Now().UTC().Format(format)
}

func (ctx StaticCtx) trim(s string, args ...string) string {
	if len(args) == 0 {
		return strings.TrimSpace(s)
	}
	return strings.Trim(s, args[0])
}

func (ctx StaticCtx) base64Encode(plain string) string {
	return base64.StdEncoding.EncodeToString([]byte(plain))
}

func (ctx StaticCtx) base64Decode(encoded string) string {
	plain, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}
	return string(plain)
}

func (ctx StaticCtx) add(a, b interface{}) interface{} {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	if ctx.isFloat(av) || ctx.isFloat(bv) {
		return ctx.reflectToFloat(av) + ctx.reflectToFloat(bv)
	}
	if ctx.isInt(av) {
		return av.Int() + ctx.reflectToInt(bv)
	}
	if ctx.isUint(av) {
		return av.Uint() + ctx.reflectToUint(bv)
	}

	return 0
}

func (ctx StaticCtx) sub(a, b interface{}) interface{} {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	if ctx.isFloat(av) || ctx.isFloat(bv) {
		return ctx.reflectToFloat(av) - ctx.reflectToFloat(bv)
	}
	if ctx.isInt(av) {
		return av.Int() - ctx.reflectToInt(bv)
	}
	if ctx.isUint(av) {
		return av.Uint() - ctx.reflectToUint(bv)
	}

	return 0
}

func (ctx StaticCtx) mult(a, b interface{}) interface{} {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	if ctx.isFloat(av) || ctx.isFloat(bv) {
		return ctx.reflectToFloat(av) * ctx.reflectToFloat(bv)
	}
	if ctx.isInt(av) {
		return av.Int() * ctx.reflectToInt(bv)
	}
	if ctx.isUint(av) {
		return av.Uint() * ctx.reflectToUint(bv)
	}

	return 0
}

func (ctx StaticCtx) div(a, b interface{}) interface{} {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	if ctx.isFloat(av) || ctx.isFloat(bv) {
		return ctx.reflectToFloat(av) / ctx.reflectToFloat(bv)
	}
	if ctx.isInt(av) {
		return av.Int() / ctx.reflectToInt(bv)
	}
	if ctx.isUint(av) {
		return av.Uint() / ctx.reflectToUint(bv)
	}

	return 0
}

func (ctx StaticCtx) parseBool(str string) bool {
	val, _ := strconv.ParseBool(str)
	return val
}

func (ctx StaticCtx) parseFloat(str string) float64 {
	val, _ := strconv.ParseFloat(str, 64)
	return val
}

func (ctx StaticCtx) parseInt(str string, args ...int) int64 {
	base := 10
	if len(args) > 0 {
		base = args[0]
	}
	val, _ := strconv.ParseInt(str, base, 64)
	return val
}

func (ctx StaticCtx) parseUint(str string, args ...int) uint64 {
	base := 10
	if len(args) > 0 {
		base = args[0]
	}
	val, _ := strconv.ParseUint(str, base, 64)
	return val
}

func (ctx StaticCtx) humanSize(size interface{}) string {
	v := reflect.ValueOf(size)
	return units.HumanSize(ctx.reflectToFloat(v))
}

func (ctx StaticCtx) reflectToFloat(val reflect.Value) float64 {
	if ctx.isFloat(val) {
		return val.Float()
	}
	if ctx.isInt(val) {
		return float64(val.Int())
	}
	if ctx.isUint(val) {
		return float64(val.Uint())
	}

	return 0
}

func (ctx StaticCtx) reflectToInt(val reflect.Value) int64 {
	if ctx.isFloat(val) {
		return int64(val.Float())
	}
	if ctx.isInt(val) || ctx.isUint(val) {
		return val.Int()
	}

	return 0
}

func (ctx StaticCtx) reflectToUint(val reflect.Value) uint64 {
	if ctx.isFloat(val) {
		return uint64(val.Float())
	}
	if ctx.isInt(val) || ctx.isUint(val) {
		return val.Uint()
	}

	return 0
}

func (ctx StaticCtx) isFloat(val reflect.Value) bool {
	kind := val.Kind()
	return kind == reflect.Float32 || kind == reflect.Float64
}

func (ctx StaticCtx) isInt(val reflect.Value) bool {
	kind := val.Kind()
	return kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64
}

func (ctx StaticCtx) isUint(val reflect.Value) bool {
	kind := val.Kind()
	return kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64
}

// kubeSeal will use the same encryption techniques as the kubeseal application found at
// https://github.com/bitnami-labs/sealed-secrets
// This function simply returns the encrypted value that can be written into a kind: SealedSecret
// resource, but it does not create the entire resource. That's left to the application developer.
func (ctx StaticCtx) kubeSeal(certData string, namespace string, name string, value string) (string, error) {
	certs, err := certUtil.ParseCertsPEM([]byte(certData))
	if err != nil {
		return "", errors.Wrap(err, "failed to parse cert")
	}

	if len(certs) == 0 {
		return "", errors.New("unable to find cert")
	}

	pubKey, ok := certs[0].PublicKey.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("failed to get public key")
	}

	rnd := rand.Reader
	sessionKey := make([]byte, 32)

	if _, err := io.ReadFull(rnd, sessionKey); err != nil {
		return "", errors.Wrap(err, "failed to read random")
	}

	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return "", errors.Wrap(err, "failed to create cipher")
	}

	aed, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.Wrap(err, "failed to create galois cipher")
	}

	// TODO consider options for clusterwide and namespacewide sealed secrets
	// But this currently only supports creation of a single type of a sealed secret
	label := []byte(fmt.Sprintf("%s/%s", namespace, name))
	rsaCiphertext, err := rsa.EncryptOAEP(sha256.New(), rnd, pubKey, sessionKey, label)
	if err != nil {
		return "", errors.Wrap(err, "failed to encrypt")
	}

	cipherText := make([]byte, 2)
	binary.BigEndian.PutUint16(cipherText, uint16(len(rsaCiphertext)))
	cipherText = append(cipherText, rsaCiphertext...)

	zeroNonce := make([]byte, aed.NonceSize())

	cipherText = aed.Seal(cipherText, zeroNonce, []byte(value), nil)

	encodedCipherText := base64.StdEncoding.EncodeToString(cipherText)
	return encodedCipherText, nil
}

func (ctx StaticCtx) namespace() string {
	// this is really only useful when called via the ffi function from kotsadm
	// because that namespace is not configurable otherwise
	if os.Getenv("DEV_NAMESPACE") != "" {
		return os.Getenv("DEV_NAMESPACE")
	}

	return util.PodNamespace
}

func (ctx StaticCtx) tlsCert(certName string, cn string, ips []interface{}, alternateDNS []interface{}, daysValid int) string {
	key := fmt.Sprintf("%s:%s", certName, cn)
	if p, ok := tlsMap[key]; ok {
		return p.Cert
	}

	p := genSelfSignedCert(cn, ips, alternateDNS, daysValid)
	tlsMap[key] = p
	tlsMap[certName] = p // backwards compatibility for tlsKey without cn argument
	return p.Cert
}

func (ctx StaticCtx) tlsKey(certName string, args ...interface{}) string {
	if len(args) != 4 {
		if p, ok := tlsMap[certName]; ok {
			return p.Key
		}
		return ""
	}

	cn, ok := args[0].(string)
	if !ok {
		return ""
	}

	ips, ok := args[1].([]interface{})
	if args[1] != nil && !ok {
		return ""
	}

	alternateDNS, ok := args[2].([]interface{})
	if args[2] != nil && !ok {
		return ""
	}

	daysValid, ok := args[3].(int)
	if !ok {
		return ""
	}

	key := fmt.Sprintf("%s:%s", certName, cn)
	if p, ok := tlsMap[key]; ok {
		return p.Key
	}

	p := genSelfSignedCert(cn, ips, alternateDNS, daysValid)
	tlsMap[key] = p
	return p.Key
}

func (ctx StaticCtx) tlsCaCert(caName string, daysValid int) string {
	cap, ok := caMap[caName]
	if !ok {
		cap = genCa(caName, daysValid)
		caMap[caName] = cap
	}

	return cap.Cert
}

func (ctx StaticCtx) tlsCertFromCa(caName, certName, cn string, ips, alternateDNS []interface{}, daysValid int) string {
	key := fmt.Sprintf("%s:%s:%s", caName, certName, cn)
	if p, ok := tlsMap[key]; ok {
		return p.Cert
	}

	p := genSignedCert(caName, cn, ips, alternateDNS, daysValid)
	tlsMap[key] = p
	return p.Cert
}

func (ctx StaticCtx) tlsKeyFromCa(caName, certName, cn string, ips, alternateDNS []interface{}, daysValid int) string {
	key := fmt.Sprintf("%s:%s:%s", caName, certName, cn)
	if p, ok := tlsMap[key]; ok {
		return p.Key
	}

	p := genSignedCert(caName, cn, ips, alternateDNS, daysValid)
	tlsMap[key] = p
	return p.Key
}

func genCa(cn string, daysValid int) TLSPair {
	tmplate := `cert: {{ $i := genCA %q %d }}{{ $i.Cert | b64enc }}
key: {{ $i.Key | b64enc }}`
	return genCertAndKey(cn, fmt.Sprintf(tmplate, cn, daysValid))
}

func genSignedCert(ca, cn string, ips []interface{}, alternateDNS []interface{}, daysValid int) TLSPair {
	tmplate := `cert: {{ $ca := buildCustomCert %q %q }}{{ $i := genSignedCert %q %s %s %d $ca }}{{ $i.Cert | b64enc }}
key: {{ $i.Key | b64enc }}`

	cap, ok := caMap[ca]
	if !ok {
		cap = genCa(ca, daysValid)
		caMap[ca] = cap
	}

	caCert := base64.StdEncoding.EncodeToString([]byte(cap.Cert))
	caKey := base64.StdEncoding.EncodeToString([]byte(cap.Key))
	ipList := arrayToTemplateList(ips)
	nameList := arrayToTemplateList(alternateDNS)
	return genCertAndKey(cn, fmt.Sprintf(tmplate, caCert, caKey, cn, ipList, nameList, daysValid))
}

func genSelfSignedCert(cn string, ips []interface{}, alternateDNS []interface{}, daysValid int) TLSPair {
	tmplate := `cert: {{ $i := genSelfSignedCert %q %s %s %d }}{{ $i.Cert | b64enc }}
key: {{ $i.Key | b64enc }}`

	ipList := arrayToTemplateList(ips)
	nameList := arrayToTemplateList(alternateDNS)
	return genCertAndKey(cn, fmt.Sprintf(tmplate, cn, ipList, nameList, daysValid))
}

func genCertAndKey(cn, templated string) TLSPair {
	parsed, err := template.New("cn").Funcs(sprig.GenericFuncMap()).Parse(templated)
	if err != nil {
		fmt.Printf("Failed to evaluate cert template: %v\n", err)
		return TLSPair{}
	}

	var buff bytes.Buffer
	if err = parsed.Execute(&buff, nil); err != nil {
		fmt.Printf("Failed to execute cert template: %v\n", err)
		return TLSPair{}
	}

	result := TLSPair{Cn: cn}
	if err := yaml.Unmarshal(buff.Bytes(), &result); err != nil {
		fmt.Printf("Failed to unmarshal cert template result: %v\n", err)
		return TLSPair{}
	}

	cert, err := base64.StdEncoding.DecodeString(result.Cert)
	if err != nil {
		fmt.Printf("Failed to decode generated cert: %v\n", err)
		return TLSPair{}
	}
	result.Cert = string(cert)

	key, err := base64.StdEncoding.DecodeString(result.Key)
	if err != nil {
		fmt.Printf("Failed to decode generated key: %v\n", err)
		return TLSPair{}
	}
	result.Key = string(key)

	return result
}

func arrayToTemplateList(items []interface{}) string {
	s := "(list"
	for _, i := range items {
		s = s + fmt.Sprintf(" %q", i)
	}
	s = s + ")"
	return s
}

// checks if this is running in a kurl cluster, by checking for the existence of a configmap 'kurl-config'
func (ctx StaticCtx) isKurl() bool {
	clientset, err := ctx.getClientset()
	if err != nil {
		return false
	}
	isKurl, _ := kurl.IsKurl(clientset)
	return isKurl
}

func getNodes(clientset kubernetes.Interface) ([]corev1.Node, error) {
	nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodeList.Items, nil
}

func (ctx StaticCtx) distribution() string {
	clientset, err := ctx.getClientset()
	if err != nil {
		return ""
	}

	// detecting openshift before getting nodes for this to work in minimal rbac
	if k8sutil.IsOpenShift(clientset) {
		return "openShift"
	}

	nodes, err := getNodes(clientset)
	if err != nil {
		return ""
	}

	_, provider := analyze.ParseNodesForProviders(nodes)

	return provider
}

func (ctx StaticCtx) nodeCount() int {
	clientset, err := ctx.getClientset()
	if err != nil {
		return 0
	}
	nodes, err := getNodes(clientset)
	if err != nil {
		return 0
	}
	return len(nodes)
}

func (ctx StaticCtx) httpsProxy() string {
	return os.Getenv("HTTPS_PROXY")
}

func (ctx StaticCtx) httpProxy() string {
	return os.Getenv("HTTP_PROXY")
}

func (ctx StaticCtx) noProxy() string {
	return os.Getenv("NO_PROXY")
}

func (ctx StaticCtx) kotsVersion() string {
	return strings.TrimPrefix(buildversion.Version(), "v")
}

func (ctx StaticCtx) yamlEscape(plain string) string {
	marshalled, err := yaml.Marshal(plain)
	if err != nil {
		return ""
	}

	// it is possible for this function to produce multiline yaml, so we indent it a bunch for safety
	indented := indent(20, string(marshalled))
	return indented
}

func (ctx StaticCtx) kubernetesVersion() string {
	clientset, err := ctx.getClientset()
	if err != nil {
		// this is so that the linter doesn't complain about semver comparisons when running outside of a k8s cluster
		return "0.0.0+unknown"
	}
	sv, err := getK8sServerVersion(clientset)
	if err != nil {
		// this is so that the linter doesn't complain about semver comparisons when running outside of a k8s cluster
		return "0.0.0+unknown"
	}
	return strings.TrimPrefix(sv.GitVersion, "v")
}

func (ctx StaticCtx) kubernetesMajorVersion() string {
	clientset, err := ctx.getClientset()
	if err != nil {
		return ""
	}
	sv, err := getK8sServerVersion(clientset)
	if err != nil {
		return ""
	}
	return sv.Major
}

func (ctx StaticCtx) kubernetesMinorVersion() string {
	clientset, err := ctx.getClientset()
	if err != nil {
		return ""
	}
	sv, err := getK8sServerVersion(clientset)
	if err != nil {
		return ""
	}
	return sv.Minor
}

func getK8sServerVersion(clientset kubernetes.Interface) (*k8sversion.Info, error) {
	sv, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes server version")
	}
	return sv, nil
}

func (ctx StaticCtx) getClientset() (kubernetes.Interface, error) {
	if ctx.clientset != nil {
		return ctx.clientset, nil
	}
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes clientset")
	}
	return clientset, nil
}

// copied from sprig
func indent(spaces int, v string) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.Replace(v, "\n", "\n"+pad, -1)
}

// use the lookup function from helm to mimic the behavior of the lookup function in helm.
func (ctx StaticCtx) lookup(apiversion string, resource string, namespace string, name string) map[string]interface{} {
	config, err := k8sutil.GetClusterConfig()
	if err != nil {
		fmt.Printf("Failed to get cluster config: %v\n", err)
		return map[string]interface{}{}
	}
	lookupFunc := helmengine.NewLookupFunction(config)
	obj, err := lookupFunc(apiversion, resource, namespace, name)
	if err != nil {
		fmt.Printf("Failed to lookup %s/%s/%s: %v\n", apiversion, resource, name, err)
		return map[string]interface{}{}
	}
	return obj
}

func (ctx StaticCtx) privateCACert() string {
	// return the name of a configmap holding additional CA certificates provided by the end user at install time
	return os.Getenv("SSL_CERT_CONFIGMAP")
}
