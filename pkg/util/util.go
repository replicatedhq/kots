package util

import (
	"bytes"
	"math/rand"
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
	var outRunes []rune
	for i := 0; i < length; i++ {
		outRunes = append(outRunes, passwordLetters[rand.Intn(len(passwordLetters))])
	}
	return string(outRunes)
}
