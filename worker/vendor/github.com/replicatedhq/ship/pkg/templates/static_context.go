package templates

/*
  This was taken from https://github.com/replicatedhq/replicated/blob/master/templates/context.go
*/

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	units "github.com/docker/go-units"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	certUtil "k8s.io/client-go/util/cert"
)

func (bb *BuilderBuilder) NewStaticContext() *StaticCtx {
	return &StaticCtx{
		Logger: bb.Logger,
	}
}

// deprecated
func NewStaticContext() *StaticCtx {
	staticCtx := &StaticCtx{
		Logger: log.NewLogfmtLogger(os.Stderr),
	}
	return staticCtx
}

type Ctx interface {
	FuncMap() template.FuncMap
}

type StaticCtx struct {
	Logger log.Logger
}

func (ctx StaticCtx) FuncMap() template.FuncMap {
	sprigMap := sprig.TxtFuncMap()

	sprigMap["Now"] = ctx.now
	sprigMap["NowFmt"] = ctx.nowFormat
	sprigMap["ToLower"] = strings.ToLower
	sprigMap["ToUpper"] = strings.ToUpper
	sprigMap["TrimSpace"] = strings.TrimSpace
	sprigMap["Trim"] = ctx.trim
	sprigMap["UrlEncode"] = url.QueryEscape
	sprigMap["Base64Encode"] = ctx.base64Encode
	sprigMap["Base64Decode"] = ctx.base64Decode
	sprigMap["Split"] = strings.Split
	sprigMap["RandomString"] = ctx.RandomString
	sprigMap["Add"] = ctx.add
	sprigMap["Sub"] = ctx.sub
	sprigMap["Mult"] = ctx.mult
	sprigMap["Div"] = ctx.div
	sprigMap["ParseBool"] = ctx.parseBool
	sprigMap["ParseFloat"] = ctx.parseFloat
	sprigMap["ParseInt"] = ctx.parseInt
	sprigMap["ParseUint"] = ctx.parseUint
	sprigMap["HumanSize"] = ctx.humanSize
	sprigMap["KubeSeal"] = ctx.kubeSeal

	return sprigMap
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
		level.Error(ctx.Logger).Log("msg", "unable to base64 decode", "err", err)
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
	level.Error(ctx.Logger).Log("msg", "unable to add")
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
	level.Error(ctx.Logger).Log("msg", "unable to sub")
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
	level.Error(ctx.Logger).Log("msg", "unable to mult")
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
	level.Error(ctx.Logger).Log("msg", "unable to div")
	return 0
}

func (ctx StaticCtx) parseBool(str string) bool {
	val, err := strconv.ParseBool(str)
	if err != nil {
		level.Error(ctx.Logger).Log("msg", "unable to parseBool", "err", err)
	}
	return val
}

func (ctx StaticCtx) parseFloat(str string) float64 {
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		level.Error(ctx.Logger).Log("msg", "unable to parseFloat", "err", err)
	}
	return val
}

func (ctx StaticCtx) parseInt(str string, args ...int) int64 {
	base := 10
	if len(args) > 0 {
		base = args[0]
	}
	val, err := strconv.ParseInt(str, base, 64)
	if err != nil {
		level.Error(ctx.Logger).Log("msg", "unable to parseInt", "err", err)
	}
	return val
}

func (ctx StaticCtx) parseUint(str string, args ...int) uint64 {
	base := 10
	if len(args) > 0 {
		base = args[0]
	}
	val, err := strconv.ParseUint(str, base, 64)
	if err != nil {
		level.Error(ctx.Logger).Log("msg", "unable to parseUint", "err", err)
	}
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
	level.Error(ctx.Logger).Log("msg", "unable to convert to float")
	return 0
}

func (ctx StaticCtx) reflectToInt(val reflect.Value) int64 {
	if ctx.isFloat(val) {
		return int64(val.Float())
	}
	if ctx.isInt(val) || ctx.isUint(val) {
		return val.Int()
	}
	level.Error(ctx.Logger).Log("msg", "unable to convert to int")
	return 0
}

func (ctx StaticCtx) reflectToUint(val reflect.Value) uint64 {
	if ctx.isFloat(val) {
		return uint64(val.Float())
	}
	if ctx.isInt(val) || ctx.isUint(val) {
		return val.Uint()
	}
	level.Error(ctx.Logger).Log("msg", "unable to convert to uint")
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
		level.Error(ctx.Logger).Log("msg", "failed to parse cert", "err", err)
		return "", err
	}

	if len(certs) == 0 {
		err := errors.New("unable to find cert in supplied cert data")
		level.Error(ctx.Logger).Log("msg", "failed to parse cert", "err", err)
		return "", err
	}

	pubKey, ok := certs[0].PublicKey.(*rsa.PublicKey)
	if !ok {
		err := fmt.Errorf("cert was not a public key, was %T", certs[0].PublicKey)
		level.Error(ctx.Logger).Log("msg", "failed to get public key", "err", err)
		return "", err
	}

	rnd := rand.Reader
	sessionKey := make([]byte, 32)

	if _, err := io.ReadFull(rnd, sessionKey); err != nil {
		return "", err
	}

	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return "", err
	}

	aed, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// TODO consider options for clusterwide and namespacewide sealed secrets
	// But this currently only supports creation of a single type of a sealed secret
	label := []byte(fmt.Sprintf("%s/%s", namespace, name))
	rsaCiphertext, err := rsa.EncryptOAEP(sha256.New(), rnd, pubKey, sessionKey, label)
	if err != nil {
		return "", err
	}

	cipherText := make([]byte, 2)
	binary.BigEndian.PutUint16(cipherText, uint16(len(rsaCiphertext)))
	cipherText = append(cipherText, rsaCiphertext...)

	zeroNonce := make([]byte, aed.NonceSize())

	cipherText = aed.Seal(cipherText, zeroNonce, []byte(value), nil)

	encodedCipherText := base64.StdEncoding.EncodeToString(cipherText)
	return encodedCipherText, nil
}
