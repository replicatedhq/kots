package s3pg

import (
	"database/sql"
	"encoding/base64"
	"os"

	"github.com/pkg/errors"
	registrytypes "github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	"go.uber.org/zap"
)

func (s S3PGStore) GetRegistryDetailsForApp(appID string) (*registrytypes.RegistrySettings, error) {
	db := persistence.MustGetPGSession()
	query := `select registry_hostname, registry_username, registry_password_enc, namespace from app where id = $1`
	row := db.QueryRow(query, appID)

	var registryHostname sql.NullString
	var registryUsername sql.NullString
	var registryPasswordEnc sql.NullString
	var registryNamespace sql.NullString

	if err := row.Scan(&registryHostname, &registryUsername, &registryPasswordEnc, &registryNamespace); err != nil {
		return nil, errors.Wrap(err, "failed to scan registry")
	}

	if !registryHostname.Valid {
		return nil, nil
	}

	registrySettings := registrytypes.RegistrySettings{
		Hostname:    registryHostname.String,
		Username:    registryUsername.String,
		PasswordEnc: registryPasswordEnc.String,
		Namespace:   registryNamespace.String,
	}

	apiCipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load apiCipher")
	}

	decodedPassword, err := base64.StdEncoding.DecodeString(registrySettings.PasswordEnc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode")
	}

	decryptedPassword, err := apiCipher.Decrypt([]byte(decodedPassword))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decrypt")
	}

	registrySettings.Password = string(decryptedPassword)

	return &registrySettings, nil
}

func (s S3PGStore) UpdateRegistry(appID string, hostname string, username string, password string, namespace string) error {
	logger.Debug("updating app registry",
		zap.String("appID", appID))

	db := persistence.MustGetPGSession()

	if password == registrytypes.PasswordMask {
		// password unchanged - don't update it
		query := `update app set registry_hostname = $1, registry_username = $2, namespace = $3 where id = $4`
		_, err := db.Exec(query, hostname, username, namespace, appID)
		if err != nil {
			return errors.Wrap(err, "failed to update registry settings")
		}
	} else {
		cipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
		if err != nil {
			return errors.Wrap(err, "failed to create aes cipher")
		}

		passwordEnc := base64.StdEncoding.EncodeToString(cipher.Encrypt([]byte(password)))

		query := `update app set registry_hostname = $1, registry_username = $2, registry_password_enc = $3, namespace = $4 where id = $5`
		_, err = db.Exec(query, hostname, username, passwordEnc, namespace, appID)
		if err != nil {
			return errors.Wrap(err, "failed to update registry settings")
		}
	}

	return nil
}
