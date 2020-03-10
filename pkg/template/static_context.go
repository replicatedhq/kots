package template

/*
  This was taken from https://github.com/replicatedhq/replicated/blob/master/templates/context.go
*/

import (
	"bytes"
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
	analyze "github.com/replicatedhq/troubleshoot/pkg/analyze"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	discovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	certUtil "k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Ctx interface {
	FuncMap() template.FuncMap
}

type StaticCtx struct {
}

type TLSPair struct {
	Cert string
	Key  string
}

var tlsMap = map[string]TLSPair{}

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

	funcMap["IsKurl"] = ctx.isKurl
	funcMap["Distribution"] = ctx.distribution

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

	return os.Getenv("POD_NAMESPACE")
}

func (ctx StaticCtx) tlsCert(certName string, cn string, ips []interface{}, alternateDNS []interface{}, daysValid int) string {
	if p, ok := tlsMap[certName]; ok {
		return p.Cert
	}

	p := genSelfSignedCert(cn, ips, alternateDNS, daysValid)
	tlsMap[certName] = p
	return p.Cert
}

func (ctx StaticCtx) tlsKey(certName string) string {
	if p, ok := tlsMap[certName]; ok {
		return p.Key
	}
	return ""
}

func genSelfSignedCert(cn string, ips []interface{}, alternateDNS []interface{}, daysValid int) TLSPair {
	tmplate := `cert: {{ $i := genSelfSignedCert %q %s %s %d }}{{ $i.Cert | b64enc }}
key: {{ $i.Key | b64enc }}`

	ipList := arrayToTemplateList(ips)
	nameList := arrayToTemplateList(alternateDNS)
	templated := fmt.Sprintf(tmplate, cn, ipList, nameList, daysValid)

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

	result := TLSPair{}
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

const kurlConfigMapName = "kurl-config"
const kurlConfigMapNamespace = "kube-system"

// checks if this is running in a kurl cluster, by checking for the existence of a configmap 'kurl-config'
func (ctx StaticCtx) isKurl() bool {
	cfg, err := config.GetConfig()
	if err != nil {
		return false
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return false
	}

	configMap, err := clientset.CoreV1().ConfigMaps(kurlConfigMapNamespace).Get(kurlConfigMapName, metav1.GetOptions{})
	if err != nil {
		return false
	}

	return configMap != nil
}

func (ctx StaticCtx) distribution() string {
	cfg, err := config.GetConfig()
	if err != nil {
		return ""
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return ""
	}

	nodeList, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return ""
	}
	nodes := nodeList.Items

	foundProviders, workingProvider := analyze.ParseNodesForProviders(nodes)

	if workingProvider != "" {
		return workingProvider
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return ""
	}

	_, apiResourceList, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		return ""
	}

	provider := analyze.CheckOpenShift(&foundProviders, apiResourceList, workingProvider)

	return provider
}
