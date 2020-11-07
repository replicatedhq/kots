package ocistore

import (
	"context"
	"database/sql"
	"encoding/base64"
	"os"

	"github.com/ocidb/ocidb/pkg/ocidb"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/kotsadm/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/crypto"
	"go.uber.org/zap"
)

func (s OCIStore) GetRegistryDetailsForApp(appID string) (*registrytypes.RegistrySettings, error) {
	query := `select registry_hostname, registry_username, registry_password_enc, namespace from app where id = $1`
	row := s.connection.DB.QueryRow(query, appID)

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

	return &registrySettings, nil
}

func (s OCIStore) UpdateRegistry(appID string, hostname string, username string, password string, namespace string) error {
	logger.Debug("updating app registry",
		zap.String("appID", appID))

	if password == registrytypes.PasswordMask {
		// password unchanged - don't update it
		query := `update app set registry_hostname = $1, registry_username = $2, namespace = $3 where id = $4`
		_, err := s.connection.DB.Exec(query, hostname, username, namespace, appID)
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
		_, err = s.connection.DB.Exec(query, hostname, username, passwordEnc, namespace, appID)
		if err != nil {
			return errors.Wrap(err, "failed to update registry settings")
		}
	}

	if err := ocidb.Commit(context.TODO(), s.connection); err != nil {
		return errors.Wrap(err, "failed to commit")
	}

	return nil
}
