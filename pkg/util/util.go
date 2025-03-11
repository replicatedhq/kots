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

func ReplicatedAPIEndpoint(license *kotsv1beta1.License) (string, error) {
	if IsEmbeddedCluster() {
		if endpoint := os.Getenv("REPLICATED_API_ENDPOINT"); endpoint != "" {
			return maybePrependHTTPS(endpoint), nil
		}
		return "", errors.New("REPLICATED_API_ENDPOINT environment variable is required")
	}

	if license != nil && license.Spec.Endpoint != "" {
		return maybePrependHTTPS(license.Spec.Endpoint), nil
	}
	return DefaultReplicatedAPIEndpoint(), nil
}

func maybePrependHTTPS(endpoint string) string {
	if !strings.HasPrefix(endpoint, "http") && !strings.HasPrefix(endpoint, "https") {
		return fmt.Sprintf("https://%s", endpoint)
	}
	return endpoint
}

func DefaultReplicatedAPIEndpoint() string {
	return "https://replicated.app"
}

func ProxyRegistryDomain(license *kotsv1beta1.License) (string, error) {
	if IsEmbeddedCluster() {
		if domain := os.Getenv("PROXY_REGISTRY_DOMAIN"); domain != "" {
			return domain, nil
		}
		return "", errors.New("PROXY_REGISTRY_DOMAIN environment variable is required")
	}

	if license != nil {
		u, err := url.Parse(license.Spec.Endpoint)
		if err == nil && u.Hostname() == "staging.replicated.app" {
			return "proxy.staging.replicated.com", nil
		}
	}

	return DefaultProxyRegistryDomain(), nil
}

func DefaultProxyRegistryDomain() string {
	return "proxy.replicated.com"
}

func ReplicatedRegistryDomain(license *kotsv1beta1.License) (string, error) {
	if license != nil {
		u, err := url.Parse(license.Spec.Endpoint)
		if err == nil && u.Hostname() == "staging.replicated.app" {
			return "registry.staging.replicated.com", nil
		}
	}

	return DefaultReplicatedRegistryDomain(), nil
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
