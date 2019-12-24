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

func NewAESCipher() (*AESCipher, error) {
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

func AESCipherFromString(data string) (aesCipher *AESCipher, initErr error) {
	defer func() {
		if r := recover(); r != nil {
			initErr = errors.Errorf("cipher init recovered from panic: %v", r)
		}
	}()

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		initErr = errors.Wrap(err, "failed to decode string")
		return
	}

	if len(decoded) < keyLength {
		initErr = errors.Errorf("cipher key is invalid: len=%d, expected %d", len(decoded), keyLength)
		return
	}

	key := decoded[:keyLength]
	block, err := aes.NewCipher(key)
	if err != nil {
		initErr = errors.Wrap(err, "failed to create cipher from data")
		return
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		initErr = errors.Wrap(err, "failed to wrap cipher gcm")
		return
	}

	nonceLen := len(decoded) - keyLength
	if nonceLen < gcm.NonceSize() {
		initErr = errors.Errorf("cipher nonce is invalid: len=%d, expected %d", nonceLen, gcm.NonceSize())
		return
	}

	aesCipher = &AESCipher{
		key:    key,
		cipher: gcm,
		nonce:  decoded[keyLength:],
	}

	return
}

func (c *AESCipher) ToString() string {
	if c == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(append(c.key, c.nonce...))
}

func (c *AESCipher) Encrypt(in []byte) []byte {
	return c.cipher.Seal(nil, c.nonce, in, nil)
}

func (c *AESCipher) Decrypt(in []byte) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("decrypt recovered from panic: %v", r)
		}
	}()

	result, err = c.cipher.Open(nil, c.nonce, in, nil)
	return
}
