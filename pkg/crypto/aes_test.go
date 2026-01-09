package crypto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_General(t *testing.T) {
	req := require.New(t)

	encryptionCipher = nil
	decryptionCiphers = nil

	// ensure that it works the first time
	req.NoError(NewAESCipher())
	cipherKey := ToString()

	// ensure that values that are encrypted can be decrypted again
	testValue := "this is a test value"
	testEncrypted := Encrypt([]byte(testValue))
	testDecrypted, err := Decrypt(testEncrypted)
	req.NoError(err)
	req.NotEqual(testValue, string(testEncrypted))
	req.Equal(testValue, string(testDecrypted))

	// ensure that rerunning it does not cause an error or change the cipherKey
	req.NoError(NewAESCipher())
	newCipherKey := ToString()
	req.Equal(cipherKey, newCipherKey)

	testDecrypted, err = Decrypt(testEncrypted)
	req.NoError(err)
	req.Equal(testValue, string(testDecrypted))

	// ensure that string values of the cipherKey can be added
	err = InitFromString(cipherKey)
	req.NoError(err)

	// ensure that empty string values passed to InitFromString are accepted
	err = InitFromString("")
	req.NoError(err)

	// ensure that invalid (non-empty) string values passed to InitFromString fail
	err = InitFromString("not a key")
	req.Error(err)
	err = InitFromString(base64.StdEncoding.EncodeToString([]byte("still not a key")))
	req.Error(err)

	// ensure that string values of other ciphers can be added and used
	altCipher := "wwYTl3RHaCirSqx7alC/hsRQXyycHDdGZZCyNMy9R01p5czC"
	altCiphertext := "sNrI1egS1iLGesPDecd8G7WoNyE/KL7IFR6mYPzWwZLY5xCC"
	altPlaintext := "this is a test value"
	req.NotEqual(cipherKey, altCipher) // a new randomly generated cipherText should not match this one

	err = InitFromString(altCipher)
	req.NoError(err)

	altCipherBytes, err := base64.StdEncoding.DecodeString(altCiphertext)
	req.NoError(err)

	altDecrypted, err := Decrypt(altCipherBytes)
	req.NoError(err)
	req.Equal(altPlaintext, string(altDecrypted))

	// ensure that after adding a new key, encryption still works and can be decrypted
	testReEncrypted := Encrypt([]byte(testValue))
	testReDecrypted, err := Decrypt(testReEncrypted)
	req.NoError(err)
	req.Equal(testValue, string(testReDecrypted))

	// ensure original encrypted value can still be decrypted (backward compatibility)
	testDecrypted, err = Decrypt(testEncrypted)
	req.NoError(err)
	req.Equal(testValue, string(testDecrypted))
}

func Test_NoDecrypt(t *testing.T) {
	req := require.New(t)

	encryptionCipher = nil
	decryptionCiphers = nil

	out, err := Decrypt([]byte("this is a test"))
	req.Error(err)
	req.ErrorIs(err, NoDecryptionKeysErr{})
	req.Nil(out)
}

func Test_BadDecrypt(t *testing.T) {
	req := require.New(t)

	encryptionCipher = nil
	decryptionCiphers = nil

	req.NoError(NewAESCipher())

	out, err := Decrypt([]byte("this is a test"))
	req.Error(err)
	req.Equal("cipher: message authentication failed", err.Error())
	req.Nil(out)
}

func Test_NoKeyEncrypt(t *testing.T) {
	req := require.New(t)

	encryptionCipher = nil
	decryptionCiphers = nil

	out := Encrypt([]byte("this is a test"))
	decrypted, err := Decrypt(out)
	req.NoError(err)
	req.Equal([]byte("this is a test"), decrypted)
}

func Test_InitFromSecret(t *testing.T) {
	req := require.New(t)

	// wipe out all ciphers to start
	encryptionCipher = nil
	decryptionCiphers = nil

	// create a new cipher and encrypt data with it
	testString := "initializing from a secret should work"
	encryptedData := Encrypt([]byte(testString))
	originalKey := ToString()

	// wipe out all ciphers again to test loading from secret
	encryptionCipher = nil
	decryptionCiphers = nil

	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kotsadm-encryption",
				Namespace: "testns",
				Labels: map[string]string{
					"test": "test",
				},
			},
			Data: map[string][]byte{
				"encryptionKey": []byte(originalKey),
			},
		})

	// load cipher from a k8s secret
	err := InitFromSecret(clientset, "testns")
	req.NoError(err)

	// compare the new key to the old key
	loadedKey := ToString()
	req.Equal(originalKey, loadedKey)

	// ensure that decryption works
	decryptedData, err := Decrypt(encryptedData)
	req.NoError(err)
	req.Equal(testString, string(decryptedData))
}

func Test_NonceIsEncodedInCiphertext(t *testing.T) {
	req := require.New(t)

	encryptionCipher = nil
	decryptionCiphers = nil
	req.NoError(NewAESCipher())

	testValue := "test nonce encoding"
	encrypted := Encrypt([]byte(testValue))

	// Verify ciphertext is longer than plaintext (includes nonce + auth tag)
	nonceSize := encryptionCipher.cipher.NonceSize()
	req.Greater(len(encrypted), len(testValue)+nonceSize)

	// Verify decryption works
	decrypted, err := Decrypt(encrypted)
	req.NoError(err)
	req.Equal(testValue, string(decrypted))
}

func Test_UniqueNoncesPerEncryption(t *testing.T) {
	req := require.New(t)

	encryptionCipher = nil
	decryptionCiphers = nil
	req.NoError(NewAESCipher())

	testValue := "test unique nonces"
	encrypted1 := Encrypt([]byte(testValue))
	encrypted2 := Encrypt([]byte(testValue))

	// Same plaintext should produce different ciphertexts (different nonces)
	req.NotEqual(encrypted1, encrypted2)

	// Both should decrypt correctly
	decrypted1, err := Decrypt(encrypted1)
	req.NoError(err)
	req.Equal(testValue, string(decrypted1))

	decrypted2, err := Decrypt(encrypted2)
	req.NoError(err)
	req.Equal(testValue, string(decrypted2))
}

func Test_LegacyCiphertextStillDecrypts(t *testing.T) {
	req := require.New(t)

	encryptionCipher = nil
	decryptionCiphers = nil

	// Use known cipher from old implementation (lines 55-68 in existing tests)
	altCipher := "wwYTl3RHaCirSqx7alC/hsRQXyycHDdGZZCyNMy9R01p5czC"
	altCiphertext := "sNrI1egS1iLGesPDecd8G7WoNyE/KL7IFR6mYPzWwZLY5xCC"
	altPlaintext := "this is a test value"

	err := InitFromString(altCipher)
	req.NoError(err)

	altCipherBytes, err := base64.StdEncoding.DecodeString(altCiphertext)
	req.NoError(err)

	// Legacy ciphertext should still decrypt with fallback logic
	altDecrypted, err := Decrypt(altCipherBytes)
	req.NoError(err)
	req.Equal(altPlaintext, string(altDecrypted))
}
