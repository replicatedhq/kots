package kotsstore

import (
	"encoding/base64"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestKOTSStore_GetRegistryDetailsForApp(t *testing.T) {
	req := require.New(t)

	db, mock, err := sqlmock.New()
	req.NoError(err)
	defer db.Close()
	persistence.InitMockDB(db)

	req.NoError(crypto.NewAESCipher())
	registryPassword := "registry-password"
	encryptedPassword := base64.StdEncoding.EncodeToString(crypto.Encrypt([]byte(registryPassword)))

	registryQueryColumns := []string{"registry_hostname", "registry_username", "registry_password_enc", "namespace", "registry_is_readonly"}
	mock.ExpectQuery("select registry_hostname, registry_username, registry_password_enc, namespace, registry_is_readonly from app where id = .+").
		WithArgs("testappid").
		WillReturnRows(sqlmock.NewRows(registryQueryColumns).AddRow("hostname", "username", encryptedPassword, "namespace", false))

	s := KOTSStore{}

	got, err := s.GetRegistryDetailsForApp("testappid")
	req.NoError(err)
	req.Equal(types.RegistrySettings{
		Hostname:    "hostname",
		Username:    "username",
		PasswordEnc: encryptedPassword,
		Password:    registryPassword,
		Namespace:   "namespace",
		IsReadOnly:  false,
	}, got)

}
