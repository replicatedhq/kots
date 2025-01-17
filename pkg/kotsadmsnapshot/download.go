package snapshot

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/downloadrequest"
	pkgresults "github.com/vmware-tanzu/velero/pkg/util/results"
)

func DownloadRestoreResults(ctx context.Context, veleroNamespace, restoreName string) ([]types.SnapshotError, []types.SnapshotError, error) {
	r, err := DownloadRequest(ctx, veleroNamespace, velerov1.DownloadTargetKindRestoreResults, restoreName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to make download request")
	}
	defer r.Close()

	resultMap := map[string]pkgresults.Result{}
	if err := json.NewDecoder(r).Decode(&resultMap); err != nil {
		return nil, nil, errors.Wrap(err, "failed to decode restore results")
	}

	warnings, errors := []types.SnapshotError{}, []types.SnapshotError{}

	for ns, messages := range resultMap["warnings"].Namespaces {
		for _, message := range messages {
			warnings = append(warnings, types.SnapshotError{
				Title:     "Warning from Namespaced Resource",
				Message:   message,
				Namespace: ns,
			})
		}
	}

	for _, message := range resultMap["warnings"].Cluster {
		warnings = append(warnings, types.SnapshotError{
			Title:   "Warning from Cluster Resource",
			Message: message,
		})
	}

	for _, message := range resultMap["warnings"].Velero {
		warnings = append(warnings, types.SnapshotError{
			Title:   "Warning from Velero Controller",
			Message: message,
		})
	}

	for ns, messages := range resultMap["errors"].Namespaces {
		for _, message := range messages {
			errors = append(errors, types.SnapshotError{
				Title:     "Error from Namespaced Resource",
				Message:   message,
				Namespace: ns,
			})
		}
	}

	for _, message := range resultMap["errors"].Cluster {
		errors = append(errors, types.SnapshotError{
			Title:   "Error from Cluster Resource",
			Message: message,
		})
	}

	// Captures Restore Hook Errors
	for _, message := range resultMap["errors"].Velero {
		errors = append(errors, types.SnapshotError{
			Title:   "Error from Velero Controller",
			Message: message,
		})
	}

	return warnings, errors, nil
}

func DownloadRequest(ctx context.Context, veleroNamespace string, kind velerov1.DownloadTargetKind, name string) (io.ReadCloser, error) {
	kbClient, err := k8sutil.GetKubeClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get velero kube client")
	}

	pr, pw := io.Pipe()
	go func() {
		err := downloadrequest.Stream(ctx, kbClient, veleroNamespace, name, kind, pw, time.Minute, true, "")
		pw.CloseWithError(err)
	}()
	return pr, nil
}
