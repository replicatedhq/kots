package kotsstore

import (
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/persistence"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/rqlite/gorqlite"
	"go.uber.org/zap"
)

func (s *KOTSStore) GetRegistryDetailsForApp(appID string) (registrytypes.RegistrySettings, error) {
	db := persistence.MustGetDBSession()
	query := `select registry_hostname, registry_username, registry_password_enc, namespace, registry_is_readonly from app where id = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{appID},
	})
	if err != nil {
		return registrytypes.RegistrySettings{}, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return registrytypes.RegistrySettings{}, ErrNotFound
	}

	var registryHostname gorqlite.NullString
	var registryUsername gorqlite.NullString
	var registryPasswordEnc gorqlite.NullString
	var registryNamespace gorqlite.NullString
	var isReadOnly gorqlite.NullBool

	if err := rows.Scan(&registryHostname, &registryUsername, &registryPasswordEnc, &registryNamespace, &isReadOnly); err != nil {
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

	decodedPassword, err := base64.StdEncoding.DecodeString(registrySettings.PasswordEnc)
	if err != nil {
		return registrytypes.RegistrySettings{}, errors.Wrap(err, "failed to decode")
	}

	decryptedPassword, err := crypto.Decrypt([]byte(decodedPassword))
	if err != nil {
		return registrytypes.RegistrySettings{}, errors.Wrap(err, "failed to decrypt")
	}

	registrySettings.Password = string(decryptedPassword)

	return registrySettings, nil
}

func (s *KOTSStore) UpdateRegistry(appID string, hostname string, username string, password string, namespace string, isReadOnly bool) error {
	logger.Debug("updating app registry",
		zap.String("appID", appID))

	db := persistence.MustGetDBSession()

	if password == registrytypes.PasswordMask {
		// password unchanged - don't update it
		query := `update app set registry_hostname = ?, registry_username = ?, namespace = ?, registry_is_readonly = ? where id = ?`
		wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: []interface{}{hostname, username, namespace, isReadOnly, appID},
		})
		if err != nil {
			return fmt.Errorf("failed to update registry settings: %v: %v", err, wr.Err)
		}
	} else {
		passwordEnc := base64.StdEncoding.EncodeToString(crypto.Encrypt([]byte(password)))

		query := `update app set registry_hostname = ?, registry_username = ?, registry_password_enc = ?, namespace = ?, registry_is_readonly = ? where id = ?`
		wr, err := db.WriteOneParameterized(gorqlite.ParameterizedStatement{
			Query:     query,
			Arguments: []interface{}{hostname, username, passwordEnc, namespace, isReadOnly, appID},
		})
		if err != nil {
			return fmt.Errorf("failed to update registry settings: %v: %v", err, wr.Err)
		}
	}

	return nil
}

func (s *KOTSStore) GetAppIDsFromRegistry(hostname string) ([]string, error) {
	db := persistence.MustGetDBSession()
	query := `select id from app where registry_hostname = ?`
	rows, err := db.QueryOneParameterized(gorqlite.ParameterizedStatement{
		Query:     query,
		Arguments: []interface{}{hostname},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query: %v: %v", err, rows.Err)
	}

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
