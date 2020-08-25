package apiserver

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/persistence"
)

func waitForDependencies(ctx context.Context) error {
	numChecks := 0
	if !strings.HasPrefix(os.Getenv("STORAGE_BASEURI"), "docker://") {
		numChecks++
	}

	errCh := make(chan error, numChecks)

	go func() {
		if !strings.HasPrefix(os.Getenv("STORAGE_BASEURI"), "docker://") {
			errCh <- waitForPostgres(ctx)
		}
	}()

	isError := false
	for i := 0; i < numChecks; i++ {
		err := <-errCh
		if err != nil {
			log.Println(err.Error())
			isError = true
		}
	}

	if isError {
		return errors.New("failed to wait for dependencies")
	}

	return nil
}

func waitForPostgres(ctx context.Context) error {
	logger.Debug("waiting for database to be ready")

	period := 1 * time.Second // TOOD: backoff
	for {
		db := persistence.MustGetPGSession()

		// any SQL will do.  just need tables to be created.
		query := `select count(1) from app`
		row := db.QueryRow(query)

		var count int
		if err := row.Scan(&count); err == nil {
			logger.Debug("database is ready")
			return nil
		}

		select {
		case <-time.After(period):
			continue
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "failed to find valid database")
		}
	}
}
