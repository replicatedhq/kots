package crypto

import (
	"encoding/base64"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_General(t *testing.T) {
	req := require.New(t)
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

	// ensure that after adding a new key, the original key is still used for encryption and decryption
	testReEncrypted := Encrypt([]byte(testValue))
	req.Equal(string(testEncrypted), string(testReEncrypted))
	testDecrypted, err = Decrypt(testEncrypted)
	req.NoError(err)
	req.Equal(testValue, string(testDecrypted))
}
