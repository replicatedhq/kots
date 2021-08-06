package util

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"math/big"
	rand "math/rand"
	"net/url"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

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
