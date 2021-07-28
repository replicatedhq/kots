package kotsstore

import (
	"database/sql"
	"encoding/base64"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"go.uber.org/zap"
)

func (s *KOTSStore) GetRegistryDetailsForApp(appID string) (registrytypes.RegistrySettings, error) {
	db := persistence.MustGetPGSession()
	query := `select registry_hostname, registry_username, registry_password_enc, namespace, registry_is_readonly from app where id = $1`
	row := db.QueryRow(query, appID)

	var registryHostname sql.NullString
	var registryUsername sql.NullString
	var registryPasswordEnc sql.NullString
	var registryNamespace sql.NullString
	var isReadOnly sql.NullBool

	if err := row.Scan(&registryHostname, &registryUsername, &registryPasswordEnc, &registryNamespace, &isReadOnly); err != nil {
		return registrytypes.RegistrySettings{}, errors.Wrap(err, "failed to scan registry")
	}

	registrySettings := registrytypes.RegistrySettings{
		Hostname:    registryHostname.String,
		Username:    registryUsername.String,
		PasswordEnc: registryPasswordEnc.String,
		Namespace:   registryNamespace.String,
		IsReadOnly:  isReadOnly.Bool,
	}

	if !registryPasswordEnc.Valid {
		return registrySettings, nil
	}

	apiCipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
	if err != nil {
		return registrytypes.RegistrySettings{}, errors.Wrap(err, "failed to load apiCipher")
	}

	decodedPassword, err := base64.StdEncoding.DecodeString(registrySettings.PasswordEnc)
	if err != nil {
		return registrytypes.RegistrySettings{}, errors.Wrap(err, "failed to decode")
	}

	decryptedPassword, err := apiCipher.Decrypt([]byte(decodedPassword))
	if err != nil {
		return registrytypes.RegistrySettings{}, errors.Wrap(err, "failed to decrypt")
	}

	registrySettings.Password = string(decryptedPassword)

	return registrySettings, nil
}

func (s *KOTSStore) UpdateRegistry(appID string, hostname string, username string, password string, namespace string, isReadOnly bool) error {
	logger.Debug("updating app registry",
		zap.String("appID", appID))

	db := persistence.MustGetPGSession()

	if password == registrytypes.PasswordMask {
		// password unchanged - don't update it
		query := `update app set registry_hostname = $1, registry_username = $2, namespace = $3, registry_is_readonly = $4 where id = $5`
		_, err := db.Exec(query, hostname, username, namespace, isReadOnly, appID)
		if err != nil {
			return errors.Wrap(err, "failed to update registry settings")
		}
	} else {
		cipher, err := crypto.AESCipherFromString(os.Getenv("API_ENCRYPTION_KEY"))
		if err != nil {
			return errors.Wrap(err, "failed to create aes cipher")
		}

		passwordEnc := base64.StdEncoding.EncodeToString(cipher.Encrypt([]byte(password)))

		query := `update app set registry_hostname = $1, registry_username = $2, registry_password_enc = $3, namespace = $4, registry_is_readonly = $5 where id = $6`
		_, err = db.Exec(query, hostname, username, passwordEnc, namespace, isReadOnly, appID)
		if err != nil {
			return errors.Wrap(err, "failed to update registry settings")
		}
	}

	return nil
}

func (s *KOTSStore) GetAppIDsFromRegistry(hostname string) ([]string, error) {
	db := persistence.MustGetPGSession()
	query := `select id from app where registry_hostname = $1`
	rows, err := db.Query(query, hostname)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query db")
	}
	defer rows.Close()

	appIDs := []string{}
	for rows.Next() {
		var appID string
		if err := rows.Scan(&appID); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		appIDs = append(appIDs, appID)
	}

	return appIDs, nil
}
