package snapshot

import (
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"go.uber.org/zap"
)

func CreateRestore(snapshotName string) error {
	logger.Debug("creating restore",
		zap.String("snapshotName", snapshotName))

	return nil
}
