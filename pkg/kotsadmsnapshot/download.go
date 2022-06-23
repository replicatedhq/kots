package snapshot

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadmsnapshot/types"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroapiv1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroclientv1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
	pkgrestore "github.com/vmware-tanzu/velero/pkg/restore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func DownloadRestoreResults(ctx context.Context, veleroNamespace, restoreName string) ([]types.SnapshotError, []types.SnapshotError, error) {
	r, err := DownloadRequest(ctx, veleroNamespace, veleroapiv1.DownloadTargetKindRestoreResults, restoreName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to make download request")
	}
	defer r.Close()

	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create new gzip reader")
	}
	defer gr.Close()

	resultMap := map[string]pkgrestore.Result{}
	if err := json.NewDecoder(gr).Decode(&resultMap); err != nil {
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
	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	veleroClient, err := veleroclientv1.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	dr := &v1.DownloadRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:         "",
			GenerateName: "dr-",
		},
		Spec: v1.DownloadRequestSpec{
			Target: velerov1.DownloadTarget{
				Kind: kind,
				Name: name,
			},
		},
	}

	downloadRequest, err := veleroClient.DownloadRequests(veleroNamespace).Create(ctx, dr, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create download request")
	}
	defer func() {
		_ = veleroClient.DownloadRequests(veleroNamespace).Delete(context.Background(), downloadRequest.Name, metav1.DeleteOptions{})
	}()

	watcher, err := veleroClient.DownloadRequests(veleroNamespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to watch download request")
	}
	defer watcher.Stop()

	signedURL, err := watchDownloadRequestForSignedURL(ctx, watcher, downloadRequest.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get signed url")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", signedURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create get request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}

	// NOTE: it is up to the caller to close this response body
	return resp.Body, nil
}

func watchDownloadRequestForSignedURL(ctx context.Context, watcher watch.Interface, name string) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()

		case e := <-watcher.ResultChan():
			if e.Type != watch.Modified {
				continue
			}
			dr, ok := e.Object.(*v1.DownloadRequest)
			if !ok {
				continue
			}
			if dr.Name != name {
				continue
			}
			if dr.Status.DownloadURL != "" {
				return dr.Status.DownloadURL, nil
			}
		}
	}
}
