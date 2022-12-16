package crypto

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type aesCipher struct {
	key    []byte
	cipher cipher.AEAD
	nonce  []byte
}

const keyLength = 24 // 192 bit

var decryptionCiphers []*aesCipher // used to decrypt data
var encryptionCipher *aesCipher    // used to encrypt data

// add cipher from API_ENCRYPTION_KEY environment variable if it is present (and set that key to be used for encryption)
func init() {
	decryptionCiphers = []*aesCipher{}
	if os.Getenv("API_ENCRYPTION_KEY") != "" {
		envCipher, err := aesCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
		if err != nil {
			// do nothing - the secret can still be initialized from a different source
		} else {
			decryptionCiphers = append(decryptionCiphers, envCipher)
			encryptionCipher = envCipher
		}
	}
}

// InitFromSecret reads the encryption key from kubernetes and adds it to the list of decryptionCiphers, and sets this key to be used for encryption.
func InitFromSecret(clientset kubernetes.Interface, namespace string) error {
	sec, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), "kotsadm-encryption", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "get kotsadm-encryption secret")
	}

	secData, ok := sec.Data["encryptionKey"]
	if !ok {
		return fmt.Errorf("kotsadm-encryption secret in %s does not have member encryptionKey", namespace)
	}

	secCipher, err := aesCipherFromString(string(secData))
	if err != nil {
		return errors.Wrap(err, "parse kotsadm-encryption secret")
	}

	addCipher(secCipher)
	encryptionCipher = secCipher

	return nil
}

// InitFromString parses the encryption key from the provided string and adds it to the list of decryptionCiphers
func InitFromString(data string) error {
	if data == "" {
		return nil
	}

	newCipher, err := aesCipherFromString(data)
	if err != nil {
		return err
	}
	addCipher(newCipher)
	return nil
}

// check if a cipher exists in the array, if it does not then add it
func addCipher(aesCipher *aesCipher) {
	foundMatch := false
	for _, existingCipher := range decryptionCiphers {
		if bytes.Equal(existingCipher.key, aesCipher.key) && bytes.Equal(existingCipher.nonce, aesCipher.nonce) {
			foundMatch = true
		}
	}

	if !foundMatch {
		decryptionCiphers = append(decryptionCiphers, aesCipher)
	}
}

// NewAESCipher creates a new AES cipher to be used for encryption and decryption. If one already exists, it is used instead.
func NewAESCipher() error {
	if encryptionCipher != nil && len(decryptionCiphers) >= 1 {
		return nil
	}

	key := make([]byte, keyLength)
	if _, err := rand.Read(key); err != nil {
		return errors.Wrap(err, "failed to read key")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return errors.Wrap(err, "failed to create new cipher")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return errors.Wrap(err, "failed to wrap cipher gcm")
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return errors.Wrap(err, "failed to read nonce")
	}

	newCipher := &aesCipher{
		key:    key,
		cipher: gcm,
		nonce:  nonce,
	}

	addCipher(newCipher)
	encryptionCipher = newCipher
	return nil
}

func aesCipherFromString(data string) (newCipher *aesCipher, initErr error) {
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

	newCipher = &aesCipher{
		key:    key,
		cipher: gcm,
		nonce:  decoded[keyLength:],
	}

	return
}

// ToString returns a string representation of the global encryption key
func ToString() string {
	if encryptionCipher == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(append(encryptionCipher.key, encryptionCipher.nonce...))
}

func (c *aesCipher) decrypt(in []byte) (result []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("decrypt recovered from panic: %v", r)
		}
	}()

	result, err = c.cipher.Open(nil, c.nonce, in, nil)
	return
}

// Encrypt encrypts the data with the registered encryption key
func Encrypt(in []byte) []byte {
	if encryptionCipher == nil {
		_ = NewAESCipher()
	}

	return encryptionCipher.cipher.Seal(nil, encryptionCipher.nonce, in, nil)
}

// Decrypt attempts to decrypt the provided data with all registered keys
func Decrypt(in []byte) (result []byte, err error) {
	if len(decryptionCiphers) == 0 {
		return nil, NoDecryptionKeysErr{}
	}

	for _, decryptCipher := range decryptionCiphers {
		result, err = decryptCipher.decrypt(in)
		if err != nil {
			continue
		} else {
			return result, nil
		}
	}
	return nil, err
}

type NoDecryptionKeysErr struct{}

func (e NoDecryptionKeysErr) Error() string {
	return "no decryption ciphers loaded"
}
