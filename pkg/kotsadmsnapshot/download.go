package snapshot

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

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

func DownloadRestoreResults(veleroNamespace, restoreName string) ([]types.SnapshotError, []types.SnapshotError, error) {
	r, err := DownloadRequest(veleroNamespace, veleroapiv1.DownloadTargetKindRestoreResults, restoreName)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to make download request")
	}
	defer r.Close()

	resultMap := map[string]pkgrestore.Result{}
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

func DownloadRequest(veleroNamespace string, kind velerov1.DownloadTargetKind, name string) (io.ReadCloser, error) {
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

	downloadRequest, err := veleroClient.DownloadRequests(veleroNamespace).Create(context.TODO(), dr, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create download request")
	}
	defer func() {
		_ = veleroClient.DownloadRequests(veleroNamespace).Delete(context.TODO(), downloadRequest.Name, metav1.DeleteOptions{})
	}()

	watcher, err := veleroClient.DownloadRequests(veleroNamespace).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to watch download request")
	}
	defer watcher.Stop()

	// generally takes less than a second
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	signedURL, err := watchDownloadRequestForSignedURL(ctx, watcher, downloadRequest.Name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get signed url")
	}

	resp, err := http.Get(signedURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	// NOTE: it is up to the caller to close this response body

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}

	return gzipReader, nil
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
