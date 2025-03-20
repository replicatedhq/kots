package util

import (
	"bytes"
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	rand "math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func IsURL(str string) bool {
	_, err := url.ParseRequestURI(str)
	if err != nil {
		return false
	}

	return true
}

func CommonSlicePrefix(first []string, second []string) []string {
	common := []string{}

	for i, a := range first {
		if i+1 > len(second) {
			return common
		}

		if first[i] != second[i] {
			return common
		}

		common = append(common, a)
	}

	return common
}

func SplitStringOnLen(str string, maxLength int) ([]string, error) {
	if maxLength >= len(str) {
		return []string{str}, nil
	}

	work := ""
	result := []string{}

	runes := bytes.Runes([]byte(str))

	for i, r := range runes {
		work = work + string(r)
		if (i+1)%maxLength == 0 {
			result = append(result, work)
			work = ""
		} else if i+1 == len(runes) {
			result = append(result, work)
		}
	}

	return result, nil
}

func IntPointer(x int) *int64 {
	var xout int64
	xout = int64(x)
	return &xout
}

var passwordLetters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// generates a [0-9a-zA-Z] password of the specified length
func GenPassword(length int) string {
	lettersLen := big.NewInt(int64(len(passwordLetters)))

	var outRunes []rune
	for i := 0; i < length; i++ {
		cryptoRandNum, err := crand.Int(crand.Reader, lettersLen)
		var randNum int64
		if err != nil {
			// print error message and fallback to math.rand's number generator
			fmt.Printf("failed to get cryptographically random number: %v\n", err)
			randNum = int64(rand.Intn(len(passwordLetters)))
		} else {
			randNum = cryptoRandNum.Int64()
		}

		outRunes = append(outRunes, passwordLetters[randNum])
	}
	return string(outRunes)
}

// CompareStringArrays returns true if all elements in arr1 are present in arr2 and the other way around.
// it does not check for equal counts of duplicates, or for ordering.
func CompareStringArrays(arr1, arr2 []string) bool {
	for _, str1 := range arr1 {
		foundMatch := false
		for _, str2 := range arr2 {
			if str1 == str2 {
				foundMatch = true
			}
		}
		if !foundMatch {
			return false
		}
	}
	for _, str2 := range arr2 {
		foundMatch := false
		for _, str1 := range arr1 {
			if str1 == str2 {
				foundMatch = true
			}
		}
		if !foundMatch {
			return false
		}
	}
	return true
}

func ConvertToSingleDocs(doc []byte) [][]byte {
	singleDocs := [][]byte{}
	// replace all windows line endings with unix line endings
	doc = bytes.ReplaceAll(doc, []byte("\r\n"), []byte("\n"))
	docs := bytes.Split(doc, []byte("\n---\n"))
	for _, doc := range docs {
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}
		singleDocs = append(singleDocs, doc)
	}
	return singleDocs
}

type ActionableError struct {
	NoRetry bool
	Message string
}

func (e ActionableError) Error() string {
	return fmt.Sprintf("%s", e.Message)
}

func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

func IsEmbeddedCluster() bool {
	return EmbeddedClusterID() != ""
}

func EmbeddedClusterID() string {
	return os.Getenv("EMBEDDED_CLUSTER_ID")
}

func EmbeddedClusterVersion() string {
	return os.Getenv("EMBEDDED_CLUSTER_VERSION")
}

var (
	ecCustomDomainsMinVer = semver.MustParse("2.2.0")
)

func IsEmbeddedClusterCustomDomainsSupported(ver string) (bool, error) {
	sv, err := semver.NewVersion(ver)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse embedded cluster version")
	}
	build := strings.Split(sv.Metadata(), ".")

	// Not yet supported in versions less than 2.2.0
	if sv.LessThan(ecCustomDomainsMinVer) {
		// TODO: remove this once we have released 2.2.0 and just return nil
		// for production releases, the second part of the build is the k8s minor version, so "30" in "2.1.3+k8s-1.30"
		// for dev builds, the second part is much longer, so "30-51-g09f1a" in "2.1.3+k8s-1.30-51-g09f1a"
		if !sv.Equal(semver.MustParse("2.1.3")) || len(build) < 2 || len(build[1]) <= 5 {
			return false, nil
		}
	}

	return true, nil
}

// ReplicatedAPIEndpoint returns the endpoint for the replicated.app API.
func ReplicatedAppEndpoint(license *kotsv1beta1.License) string {
	if ep := os.Getenv("REPLICATED_APP_ENDPOINT"); ep != "" {
		return ep
	}

	// If this is embedded cluster, REPLICATED_APP_ENDPOINT is required.
	if IsEmbeddedCluster() {
		panic("REPLICATED_APP_ENDPOINT environment variable is required for embedded cluster")
	}

	// If this is not an embedded cluster, use the legacy behavior and fall back to the license
	// spec endpoint or the default value.

	// DEPRECATED: use REPLICATED_APP_ENDPOINT instead
	if ep := os.Getenv("REPLICATED_API_ENDPOINT"); ep != "" {
		return ep
	}

	if license != nil {
		return license.Spec.Endpoint
	}

	return "https://replicated.app"
}

func DefaultProxyRegistryDomain() string {
	return "proxy.replicated.com"
}

func DefaultReplicatedRegistryDomain() string {
	return "registry.replicated.com"
}

func IsUpgradeService() bool {
	return os.Getenv("IS_UPGRADE_SERVICE") == "true"
}

func HTTPProxy() string {
	return os.Getenv("HTTP_PROXY")
}

func HTTPSProxy() string {
	return os.Getenv("HTTPS_PROXY")
}

func NoProxy() string {
	return os.Getenv("NO_PROXY")
}

func GetValueFromMapPath(m interface{}, path []string) interface{} {
	if len(path) == 0 {
		return nil
	}

	key := path[0]
	if ms, ok := m.(map[string]interface{}); ok {
		for k, v := range ms {
			if k != key {
				continue
			}
			if len(path) == 1 {
				return v
			}
			return GetValueFromMapPath(v, path[1:])
		}
		return nil
	}

	if mi, ok := m.(map[interface{}]interface{}); ok {
		for k, v := range mi {
			if s, ok := k.(string); !ok || s != key {
				continue
			}
			if len(path) == 1 {
				return v
			}
			return GetValueFromMapPath(v, path[1:])
		}
		return nil
	}

	return nil
}

func Base64DecodeInterface(d interface{}) ([]byte, error) {
	var bytes []byte
	switch d := d.(type) {
	case string:
		bytes = []byte(d)
	case []byte:
		bytes = d
	default:
		return nil, errors.Errorf("cannot base64 decode %T", d)
	}

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(bytes)))
	n, err := base64.StdEncoding.Decode(decoded, []byte(bytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to bse64 decode interface data")
	}
	return decoded[:n], nil
}

func StrPointer(s string) *string {
	return &s
}

func GetFilesMap(dir string) (map[string][]byte, error) {
	filesMap := map[string][]byte{}

	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				return errors.Wrapf(err, "failed to read file %s", path)
			}

			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return errors.Wrapf(err, "failed to get relative path for %s", path)
			}

			filesMap[relPath] = contents

			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk dir")
	}

	return filesMap, nil
}

func DecryptConfigValue(input string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", errors.Wrap(err, "failed to base64 decode")
	}

	decrypted, err := crypto.Decrypt(decoded)
	if err != nil {
		return "", errors.Wrap(err, "failed to decrypt")
	}

	return string(decrypted), nil
}
