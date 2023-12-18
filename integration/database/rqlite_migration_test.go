package database

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"testing"

	_ "github.com/lib/pq"
	dockertest "github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/persistence"
	"github.com/rqlite/gorqlite"
)

const (
	POSTGRES_SCHEMA_DIR = "../../deploy/assets/postgres/tables"
	RQLITE_SCHEMA_DIR   = "../../migrations/tables"
	RQLITE_AUTH_CONFIG  = `[{"username": "kotsadm", "password": "password", "perms": ["all"]}, {"username": "*", "perms": ["status", "ready"]}]`
)

func TestMigrateFromPostgresToRqlite(t *testing.T) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Failed to connect to docker: %s", err)
	}

	// remove containers if present
	if err := pool.RemoveContainerByName("postgres"); err != nil {
		t.Fatalf("Failed to remove postgres container: %v", err)
	}
	if err := pool.RemoveContainerByName("rqlite"); err != nil {
		t.Fatalf("Failed to remove rqlite container: %v", err)
	}

	// start postgres db
	pgRunOptions := &dockertest.RunOptions{
		Name:       "postgres",
		Repository: "postgres",
		Tag:        "14.5-alpine",
		Env: []string{
			"POSTGRES_USER=kotsadm",
			"POSTGRES_PASSWORD=password",
			"POSTGRES_DB=kotsadm",
			"POSTGRES_HOST_AUTH_METHOD=md5",
		},
	}
	pgHostConfig := func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	}
	postgres, err := pool.RunWithOptions(pgRunOptions, pgHostConfig)
	if err != nil {
		t.Fatalf("Failed to start postgres: %s", err)
	}

	// start rqlite db
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("Failed to get current user: %s", err)
	}
	rqliteAuthConfigPath := path.Join(t.TempDir(), "rqlite-auth-config.json")
	if err := os.WriteFile(rqliteAuthConfigPath, []byte(RQLITE_AUTH_CONFIG), 0644); err != nil {
		t.Fatalf("Failed to write to file %s", rqliteAuthConfigPath)
	}
	rqliteTag, _ := image.GetTag(image.Rqlite)
	rqliteRunOptions := &dockertest.RunOptions{
		Name:       "rqlite",
		Repository: "kotsadm/rqlite",
		Tag:        rqliteTag,
		Mounts: []string{
			fmt.Sprintf("%s:/rqlite/file", t.TempDir()),
			fmt.Sprintf("%s:/auth/config.json", rqliteAuthConfigPath),
		},
		ExposedPorts: []string{
			"4001/tcp",
			"4002/tcp",
		},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"4001/tcp": {
				{
					HostIP:   "localhost",
					HostPort: "14001",
				},
			},
		},
		Cmd: []string{
			"-http-adv-addr=localhost:14001",
			"-auth=/auth/config.json",
		},
		User: currentUser.Uid,
	}
	rqliteHostConfig := func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	}
	rqlite, err := pool.RunWithOptions(rqliteRunOptions, rqliteHostConfig)
	if err != nil {
		t.Fatalf("Failed to start rqlite: %s", err)
	}

	// connection strings
	pgURI := fmt.Sprintf("postgres://kotsadm:password@localhost:%s/kotsadm?connect_timeout=10&sslmode=disable", postgres.GetPort("5432/tcp"))
	rqliteURI := fmt.Sprintf("http://kotsadm:password@localhost:%s?timeout=10", rqlite.GetPort("4001/tcp"))

	// wait for postgres to be ready
	var pgDB *sql.DB
	if err := pool.Retry(func() error {
		var err error
		pgDB, err = sql.Open("postgres", pgURI)
		if err != nil {
			return errors.Wrap(err, "failed to open postgres connection")
		}
		return pgDB.Ping()
	}); err != nil {
		log.Fatalf("Failed to connect to postgres: %s", err)
	}

	// wait for rqlite to be ready
	var rqliteDB gorqlite.Connection
	if err := pool.Retry(func() error {
		url := fmt.Sprintf("http://localhost:%s/readyz", rqlite.GetPort("4001/tcp"))
		newRequest, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create new request")
		}
		resp, err := http.DefaultClient.Do(newRequest)
		if err != nil {
			return errors.Wrap(err, "failed to do request")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errors.Errorf("unexpected status code %d", resp.StatusCode)
		}

		rqliteDB, err = gorqlite.Open(rqliteURI)
		if err != nil {
			return errors.Wrap(err, "failed to open rqlite connection")
		}
		return nil
	}); err != nil {
		log.Fatalf("Failed to connect to rqlite: %s", err)
	}

	// init dbs vars
	t.Setenv("POSTGRES_URI", pgURI)
	t.Setenv("POSTGRES_SCHEMA_DIR", POSTGRES_SCHEMA_DIR)
	persistence.SetDB(&rqliteDB)

	// update postgres schema
	if err := persistence.UpdateDBSchema("postgres", pgURI, POSTGRES_SCHEMA_DIR); err != nil {
		t.Fatalf("Failed to update postgres schema: %s", err)
	}

	// update rqlite schema
	if err := persistence.UpdateDBSchema("rqlite", rqliteURI, RQLITE_SCHEMA_DIR); err != nil {
		t.Fatalf("Failed to update rqlite schema: %s", err)
	}

	// insert data into postgres
	if err := insertDataIntoPostgres(pgDB); err != nil {
		t.Fatalf("Failed to insert data into postgres: %s", err)
	}

	// migrate data from postgres to rqlite
	if err := persistence.MigrateFromPostgresToRqlite(); err != nil {
		t.Fatalf("Failed to migrate data from postgres to rqlite: %s", err)
	}

	// validate data in rqlite
	if err := validateDataInRqlite(&rqliteDB); err != nil {
		t.Fatalf("Failed to validate data in rqlite: %s", err)
	}

	if err := pool.Purge(postgres); err != nil {
		log.Fatalf("Failed to purge postgres: %s", err)
	}
	if err := pool.Purge(rqlite); err != nil {
		log.Fatalf("Failed to purge rqlite: %s", err)
	}
}

func insertDataIntoPostgres(pgDB *sql.DB) error {
	tx, err := pgDB.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	// api_task_status
	query := `INSERT INTO api_task_status VALUES (
		'id',
		'2021-09-01 00:00:00',
		'current message',
		'complete'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into api_task_status")
	}

	// app_downstream_output
	query = `INSERT INTO app_downstream_output VALUES (
		'appid',
		'clusterid',
		'1',
		'dryrun-stdout',
		'dryrun-stderr',
		'apply-stdout',
		'apply-stderr',
		'helm-stdout',
		'helm-stderr',
		'false'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into app_downstream_output")
	}

	// app_downstream_version
	query = `INSERT INTO app_downstream_version VALUES (
		'appid',
		'clusterid',
		'1',
		'1',
		'2021-09-01 00:00:00',
		'2021-09-01 00:00:00',
		'v1.0.0',
		'complete',
		'status info',
		'upstream change',
		'diff-summary',
		'diff-summary-error',
		'preflight-progress',
		'preflight-result',
		'2021-09-01 00:00:00',
		'false',
		'false',
		'git-commit-url',
		'false'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into app_downstream_version")
	}

	// app_downstream
	query = `INSERT INTO app_downstream VALUES (
		'appid',
		'clusterid',
		'downstream-name',
		'1'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into app_downstream")
	}

	// app_status
	query = `INSERT INTO app_status VALUES (
		'appid',
		'{}',
		'2021-09-01 00:00:00',
		'1'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into app_status")
	}

	// app_version
	query = `INSERT INTO app_version VALUES (
		'appid',
		'1',
		'1',
		'channelid',
		'channelname',
		'2021-09-01 00:00:00',
		'2021-09-01 00:00:00',
		'v1.0.0',
		'false',
		'release notes',
		'supportbundle spec',
		'preflight spec',
		'analyzer spec',
		'app spec',
		'kots app spec',
		'kots installation spec',
		'kots license',
		'config spec',
		'config values',
		'2021-09-01 00:00:00',
		'status',
		'encryption key',
		'backup spec',
		'identity spec',
		decode('68656c6c6f20776f726c64', 'hex')
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into app_version")
	}

	// app
	query = `INSERT INTO app VALUES (
		'appid',
		'myapp',
		'icon-uri',
		'2021-01-01 00:00:00',
		'2021-01-01 00:00:00',
		'myapp',
		'my-app',
		'not-a-license',
		'1',
		'2021-01-01 00:00:00',
		'false',
		'registryhostname',
		'registryusername',
		'registrypassword',
		'registrypasswordenc',
		'namespace',
		'false',
		'2021-01-01 00:00:00',
		'2021-01-01 00:00:00',
		'installed',
		'false',
		'0',
		'0 0 * * *',
		NULL,
		'restore-undeploy-status',
		'0 0 * * *',
		'false',
		'false'
	)`
	_, err = tx.Exec(query)
	if err != nil {
		return errors.Wrap(err, "failed to insert into app")
	}

	// cluster
	query = `INSERT INTO cluster VALUES (
		'clusterid',
		'clustername',
		'clusterslug',
		'2021-01-01 00:00:00',
		'2021-01-01 00:00:00',
		'clustertoken',
		'clustertype',
		'false',
		'0 0 * * *',
		'720h'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into cluster")
	}

	// initial_branding
	query = `INSERT INTO initial_branding VALUES (
		'id',
		decode('68656c6c6f20776f726c64', 'hex'),
		'2021-09-01 00:00:00'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into initial_branding")
	}

	// kotsadm_params
	query = `INSERT INTO kotsadm_params VALUES (
		'somekey',
		'somevalue'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into kotsadm_params")
	}

	// object_store
	query = `INSERT INTO object_store VALUES (
		'filepath',
		'encoded_block'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into object_store")
	}

	// pending_supportbundle
	query = `INSERT INTO pending_supportbundle VALUES (
		'id',
		'appid',
		'clusterid',
		'2021-09-01 00:00:00'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into pending_supportbundle")
	}

	// preflight_result
	query = `INSERT INTO preflight_result VALUES (
		'id',
		'appid',
		'result',
		'2021-09-01 00:00:00'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into preflight_result")
	}

	// preflight_spec
	query = `INSERT INTO preflight_spec VALUES (
		'appid',
		'1',
		'preflight spec'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into preflight_spec")
	}

	// scheduled_instance_snapshots
	query = `INSERT INTO scheduled_instance_snapshots VALUES (
		'id',
		'clusterid',
		'2021-09-01 00:00:00',
		'backupname'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into scheduled_instance_snapshots")
	}

	// scheduled_snapshots
	query = `INSERT INTO scheduled_snapshots VALUES (
		'id',
		'appid',
		'2021-09-01 00:00:00',
		'backupname'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into scheduled_snapshots")
	}

	// session
	query = `INSERT INTO session VALUES (
		'id',
		'userid',
		'metadata',
		'2021-09-01 00:00:00',
		'2021-09-01 00:00:00'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into session")
	}

	// ship_user_local
	query = `INSERT INTO ship_user_local VALUES (
		'userid',
		'password_bcrypt',
		'first_name',
		'last_name',
		'email'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into ship_user_local")
	}

	// ship_user
	query = `INSERT INTO ship_user VALUES (
		'userid',
		'2021-09-01 00:00:00',
		'123',
		'2021-09-01 00:00:00'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into ship_user")
	}

	// supportbundle_analysis
	query = `INSERT INTO supportbundle_analysis VALUES (
		'id',
		'supportbundleid',
		'error',
		'maxseverity',
		'insights',
		'2021-09-01 00:00:00'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into supportbundle_analysis")
	}

	// supportbundle
	query = `INSERT INTO supportbundle VALUES (
		'id',
		'myapp',
		'appid',
		'name',
		'100',
		'pending',
		'{}',
		'analysisid',
		'2021-09-01 00:00:00',
		'2021-09-01 00:00:00',
		'2021-09-01 00:00:00',
		'false',
		'redact report'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into supportbundle")
	}

	// user_app
	query = `INSERT INTO user_app VALUES (
		'userid',
		'appid'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into user_app")
	}

	// user_cluster
	query = `INSERT INTO user_cluster VALUES (
		'userid',
		'clusterid'
	)`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "failed to insert into user_cluster")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

func validateDataInRqlite(rqliteDB *gorqlite.Connection) error {
	// api_task_status
	query := `SELECT * FROM api_task_status`
	rows, err := rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query api_task_status: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in api_task_status")
	}

	// app_downstream_output
	query = `SELECT * FROM app_downstream_output`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query app_downstream_output: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in app_downstream_output")
	}

	// app_downstream_version
	query = `SELECT * FROM app_downstream_version`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query app_downstream_version: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in app_downstream_version")
	}

	// app_downstream
	query = `SELECT * FROM app_downstream`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query app_downstream: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in app_downstream")
	}

	// app_status
	query = `SELECT * FROM app_status`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query app_status: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in app_status")
	}

	// app_version
	query = `SELECT * FROM app_version`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query app_version: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in app_version")
	}

	// app
	query = `select id, name, created_at, restore_in_progress_name from app where id = 'appid'`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query app: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.Errorf("app 'appid' not found")
	}

	var id, name string
	var createdAt int64 // int64 not time.Time to validate that it's been converted into a valid unix timestamp
	var restoreInProgressName gorqlite.NullString
	if err := rows.Scan(&id, &name, &createdAt, &restoreInProgressName); err != nil {
		return errors.Wrap(err, "failed to scan app")
	}

	if id != "appid" {
		return errors.Errorf("expected app id to be 'appid', got %q", id)
	}
	if name != "myapp" {
		return errors.Errorf("expected app name to be 'myapp', got %q", name)
	}
	if createdAt != 1609459200 {
		return errors.Errorf("expected app created_at to be 1609459200, got %d", createdAt)
	}
	if restoreInProgressName.Valid {
		return errors.Errorf("expected restore_in_progress_name to be null, got %q", restoreInProgressName.String)
	}

	// cluster
	query = `SELECT * FROM cluster`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query cluster: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in cluster")
	}

	// initial_branding
	query = `SELECT contents FROM initial_branding`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query initial_branding: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in initial_branding")
	}
	// validate that the bytea contents have been converted into a base64 encoded string
	var brandingContents string
	if err := rows.Scan(&brandingContents); err != nil {
		return errors.Wrap(err, "failed to scan initial_branding")
	}
	if brandingContents != "aGVsbG8gd29ybGQ=" {
		return errors.Errorf("expected initial_branding contents to be 'aGVsbG8gd29ybGQ=', got %q", brandingContents)
	}

	// kotsadm_params
	// this table should contain 2 records:
	// - one that was inserted by the test
	// - one that marks a successful migration
	query = `SELECT key, value FROM kotsadm_params`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query kotsadm_params: %v: %v", err, rows.Err)
	}
	if rows.NumRows() != 2 {
		return errors.Errorf("expected 2 records in kotsadm_params, found: %d", rows.NumRows())
	}

	rows.Next()
	var key, value string
	if err := rows.Scan(&key, &value); err != nil {
		return errors.Wrap(err, "failed to scan kotsadm_params")
	}
	if key != "somekey" {
		return errors.Errorf("expected key to be 'somekey', got %q", key)
	}
	if value != "somevalue" {
		return errors.Errorf("expected value of 'somekey' to be 'somevalue', got %q", value)
	}

	rows.Next()
	if err := rows.Scan(&key, &value); err != nil {
		return errors.Wrap(err, "failed to scan kotsadm_params")
	}
	if key != persistence.RQLITE_MIGRATION_SUCCESS_KEY {
		return errors.Errorf("expected key to be %q, got %q", persistence.RQLITE_MIGRATION_SUCCESS_KEY, key)
	}
	if value != persistence.RQLITE_MIGRATION_SUCCESS_VALUE {
		return errors.Errorf("expected value of %q to be %q, got %q", persistence.RQLITE_MIGRATION_SUCCESS_KEY, persistence.RQLITE_MIGRATION_SUCCESS_VALUE, value)
	}

	// object_store
	query = `SELECT * FROM object_store`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query object_store: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in object_store")
	}

	// pending_supportbundle
	query = `SELECT * FROM pending_supportbundle`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query pending_supportbundle: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in pending_supportbundle")
	}

	// preflight_result
	query = `SELECT * FROM preflight_result`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query preflight_result: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in preflight_result")
	}

	// preflight_spec
	query = `SELECT * FROM preflight_spec`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query preflight_spec: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in preflight_spec")
	}

	// scheduled_instance_snapshots
	query = `SELECT * FROM scheduled_instance_snapshots`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query scheduled_instance_snapshots: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in scheduled_instance_snapshots")
	}

	// scheduled_snapshots
	query = `SELECT * FROM scheduled_snapshots`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query scheduled_snapshots: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in scheduled_snapshots")
	}

	// session
	query = `SELECT * FROM session`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query session: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in session")
	}

	// ship_user_local
	query = `SELECT * FROM ship_user_local`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query ship_user_local: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in ship_user_local")
	}

	// ship_user
	query = `SELECT * FROM ship_user`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query ship_user: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in ship_user")
	}

	// supportbundle_analysis
	query = `SELECT * FROM supportbundle_analysis`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query supportbundle_analysis: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in supportbundle_analysis")
	}

	// supportbundle
	query = `SELECT * FROM supportbundle`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query supportbundle: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in supportbundle")
	}

	// user_app
	query = `SELECT * FROM user_app`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query user_app: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in user_app")
	}

	// user_cluster
	query = `SELECT * FROM user_cluster`
	rows, err = rqliteDB.QueryOne(query)
	if err != nil {
		return fmt.Errorf("failed to query user_cluster: %v: %v", err, rows.Err)
	}
	if !rows.Next() {
		return errors.New("no records found in user_cluster")
	}

	return nil
}
