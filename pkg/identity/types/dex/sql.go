package types

/*
Those types have been copied from dex repo because it had go mod conflicts with other packages
link: https://github.com/dexidp/dex/blob/master/storage/sql/config.go
*/

// NetworkDB contains options common to SQL databases accessed over network.
type NetworkDB struct {
	Database string
	User     string
	Password string
	Host     string
	Port     uint16

	ConnectionTimeout int // Seconds

	// database/sql tunables, see
	// https://golang.org/pkg/database/sql/#DB.SetConnMaxLifetime and below
	// Note: defaults will be set if these are 0
	MaxOpenConns    int // default: 5
	MaxIdleConns    int // default: 5
	ConnMaxLifetime int // Seconds, default: not set
}

// SSL represents SSL options for network databases.
type SSL struct {
	Mode   string
	CAFile string
	// Files for client auth.
	KeyFile  string
	CertFile string
}

// Postgres options for creating an SQL db.
type Postgres struct {
	NetworkDB

	SSL SSL `json:"ssl" yaml:"ssl"`
}
