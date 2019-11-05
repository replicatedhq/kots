package main

import "C"

import (
	"encoding/base64"
	"fmt"

	"github.com/replicatedhq/kots/pkg/crypto"
)

//export EncryptString
func EncryptString(cipherString string, message string) *C.char {
	cipher, err := crypto.AESCipherFromString(cipherString)
	if err != nil {
		fmt.Printf("failed to create cipher: %v\n", err)
		return nil
	}

	encrypted := cipher.Encrypt([]byte(message))
	return C.CString(base64.StdEncoding.EncodeToString(encrypted))
}

//export DecryptString
func DecryptString(cipherString string, messageBase64 string) *C.char {
	cipher, err := crypto.AESCipherFromString(cipherString)
	if err != nil {
		fmt.Printf("failed to create cipher: %v\n", err)
		return nil
	}

	message, err := base64.StdEncoding.DecodeString(messageBase64)
	if err != nil {
		fmt.Printf("failed to decode message: %v\n", err)
		return nil
	}

	decrypted, err := cipher.Decrypt([]byte(message))
	if err != nil {
		fmt.Printf("failed to decrypt message: %v\n", err)
		return nil
	}

	return C.CString(string(decrypted))
}
