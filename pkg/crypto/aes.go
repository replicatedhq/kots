package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"

	"github.com/pkg/errors"
)

type AESCipher struct {
	key    []byte
	cipher cipher.AEAD
	nonce  []byte
}

const keyLength = 24 // 192 bit

func NewAESCypher() (*AESCipher, error) {
	key := make([]byte, keyLength)
	if _, err := rand.Read(key); err != nil {
		return nil, errors.Wrap(err, "failed to read key")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed ro create new cipher")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap cipher gcm")
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, errors.Wrap(err, "failed to read nonce")
	}

	return &AESCipher{
		key:    key,
		cipher: gcm,
		nonce:  nonce,
	}, nil
}

func AESCipherFromString(data string) (*AESCipher, error) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode string")
	}

	if len(decoded) < keyLength {
		return nil, errors.Errorf("cipher key is invalid: len=%d, expected %d", len(decoded), keyLength)
	}

	key := decoded[:keyLength]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cipher from data")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wrap cipher gcm")
	}

	nonceLen := len(decoded) - keyLength
	if nonceLen < gcm.NonceSize() {
		return nil, errors.Errorf("cipher nonce is invalid: len=%d, expected %d", nonceLen, gcm.NonceSize())
	}

	return &AESCipher{
		key:    key,
		cipher: gcm,
		nonce:  decoded[keyLength:],
	}, nil
}

func (c *AESCipher) ToString() string {
	return base64.StdEncoding.EncodeToString(append(c.key, c.nonce...))
}
