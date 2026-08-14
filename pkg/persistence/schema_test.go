package persistence

import (
	"testing"

	schemasv1alpha4 "github.com/schemahero/schemahero/pkg/apis/schemas/v1alpha4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_driverTableSchema(t *testing.T) {
	pg := &schemasv1alpha4.PostgresqlTableSchema{}
	rq := &schemasv1alpha4.RqliteTableSchema{}

	tests := []struct {
		name    string
		driver  string
		schema  *schemasv1alpha4.TableSchema
		want    any
		wantErr string
	}{
		{
			name:   "postgres schema present",
			driver: "postgres",
			schema: &schemasv1alpha4.TableSchema{Postgres: pg},
			want:   pg,
		},
		{
			name:   "rqlite schema present",
			driver: "rqlite",
			schema: &schemasv1alpha4.TableSchema{RQLite: rq},
			want:   rq,
		},
		{
			name:    "nil schema is an error",
			driver:  "rqlite",
			schema:  nil,
			wantErr: "no schema",
		},
		{
			name:    "postgres driver without a postgres schema is an error",
			driver:  "postgres",
			schema:  &schemasv1alpha4.TableSchema{RQLite: rq},
			wantErr: "no postgres schema",
		},
		{
			name:    "rqlite driver without an rqlite schema is an error",
			driver:  "rqlite",
			schema:  &schemasv1alpha4.TableSchema{Postgres: pg},
			wantErr: "no rqlite schema",
		},
		{
			name:    "unsupported driver is an error",
			driver:  "mysql",
			schema:  &schemasv1alpha4.TableSchema{Postgres: pg},
			wantErr: "unsupported schemahero driver",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := driverTableSchema(tt.driver, tt.schema)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_connectSchemahero_unsupportedDriver(t *testing.T) {
	conn, err := connectSchemahero("mysql", "")
	require.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "unsupported schemahero driver")
}
