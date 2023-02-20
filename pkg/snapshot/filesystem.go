package snapshot

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/replicatedhq/kots/pkg/snapshot/types"
)

func GetCurrentFileSystemConfig(ctx context.Context, namespace string, isMinioDisabled bool) (*types.FileSystemConfig, error) {
	var fileSystemConfig *types.FileSystemConfig

	if !isMinioDisabled {
		fileSystemConfig, err := GetCurrentMinioFileSystemConfig(ctx, namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get current minio file system config")
		}
		if fileSystemConfig != nil {
			return fileSystemConfig, nil
		}
		return nil, nil
	}

	fileSystemConfig, err := GetCurrentLvpFileSystemConfig(ctx, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current lvp file system config")
	}

	return fileSystemConfig, nil
}
