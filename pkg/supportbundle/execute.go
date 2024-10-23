package supportbundle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/replicatedhq/troubleshoot/pkg/redact"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/supportbundle"
)

type supportBundleProgressUpdate struct {
	Message string
	Status  types.SupportBundleStatus
	Type    supportBundleProgressUpdateType
	Size    *float64
}

type supportBundleProgressUpdateType string

const (
	BUNDLE_PROGRESS_ERROR          supportBundleProgressUpdateType = "error"
	BUNDLE_PROGRESS_COLLECTOR      supportBundleProgressUpdateType = "collector"
	BUNDLE_PROGRESS_HOST_COLLECTOR supportBundleProgressUpdateType = "collect.CollectProgress"
	BUNDLE_PROGRESS_FILETREE       supportBundleProgressUpdateType = "filetree"
	BUNDLE_PROGRESS_UPLOADED       supportBundleProgressUpdateType = "uploaded"
)

// supportBundleIsComplete checks the current progress against the declared totals in
// the support bundle.
func supportBundleIsComplete(bundle *types.SupportBundle, collectorCount int) bool {
	return collectorCount == bundle.Progress.CollectorCount &&
		bundle.Status == types.BUNDLE_UPLOADED
}

// executeUpdateRoutine creates a goroutine to manage updates to the support bundle secret while processing.
// The function returns a progress channel to be used by the executor. The goroutine completes
// the executor closes the channel or the internal timeout expires.
func executeUpdateRoutine(bundle *types.SupportBundle) chan interface{} {
	progressChan := make(chan interface{})

	timeout := time.After(60 * time.Minute)
	logger.Infof("Executing Update go routine for support bundle ID: %s", bundle.ID)

	go func() {
		var collectorsComplete int
		logger.Debugf("Waiting for %d collectors to complete", bundle.Progress.CollectorCount)

		updateTicker := time.NewTicker(5 * time.Second) // without updates, bundle will be considlered failed after 10 seconds.
		defer updateTicker.Stop()

		for {
			select {
			case msg, ok := <-progressChan:
				// Closed by sender
				if !ok {
					logger.Debugf("Progress channel update closed for support bundle ID: %s", bundle.ID)

					if !supportBundleIsComplete(bundle, collectorsComplete) {
						bundleErrorUpdate(bundle)
					} else {
						logger.Debugf("Bundle Complete: %s", bundle.ID)
					}

					return
				}

				fmt.Println(msg)
				fmt.Println(reflect.TypeOf(msg))

				switch val := msg.(type) {
				case error:
					// Errors could be expected with RBAC, just log and continue
					logger.Infof("Progress channel received an error, %s, for support bundle ID: %s", val, bundle.ID)

				case supportBundleProgressUpdate:
					// Collect events are saved separately since there are many

					if val.Type == BUNDLE_PROGRESS_COLLECTOR {
						logger.Debugf("Received collector progress update %d, %s, for support bundle ID: %s", collectorsComplete, val.Message, bundle.ID)

						collectorsComplete++

						bundle.Progress.CollectorsCompleted = collectorsComplete
						bundle.Progress.Message = val.Message
						if err := store.GetStore().UpdateSupportBundle(bundle); err != nil {
							logger.Error(errors.Wrap(err, "could not update collector counter for bundle"))
							return
						}

						// Host collectors are saved separately since there are many
					} else if val.Type == BUNDLE_PROGRESS_HOST_COLLECTOR {
						logger.Debugf("Received host collector progress update %d, %s, for support bundle ID: %s", collectorsComplete, val.Message, bundle.ID)

						bundle.Progress.CollectorsCompleted = collectorsComplete
						bundle.Progress.Message = val.Message

						if err := store.GetStore().UpdateSupportBundle(bundle); err != nil {
							logger.Error(errors.Wrap(err, "could not update progress for bundle"))
							return
						}

						// Something went wrong and the loop finished
					} else if val.Type == BUNDLE_PROGRESS_ERROR {
						logger.Debugf("Received error in progress update, %s, for support bundle ID: %s", val.Message, bundle.ID)

						bundle.Progress.Message = val.Message
						bundleErrorUpdate(bundle)

						// Finished
					} else if val.Type == BUNDLE_PROGRESS_UPLOADED && val.Status == types.BUNDLE_UPLOADED {
						logger.Debugf("Received %s progress update, %s, for support bundle ID: %s", val.Type, val.Message, bundle.ID)

						now := time.Now()
						bundle.UploadedAt = &now
						bundle.Size = *val.Size
						bundle.Status = types.BUNDLE_UPLOADED
						bundle.Progress.Message = val.Message

						if err := store.GetStore().UpdateSupportBundle(bundle); err != nil {
							logger.Error(errors.Wrap(err, "could not update uploaded status for bundle"))
							return
						}

						// Generic updates
					} else {
						logger.Debugf("Received %s progress update, %s, for support bundle ID: %s", val.Type, val.Message, bundle.ID)

						bundle.Progress.Message = val.Message

						if err := store.GetStore().UpdateSupportBundle(bundle); err != nil {
							logger.Error(errors.Wrap(err, "could not update progress for bundle"))
							return
						}
					}
				default:
					logger.Errorf("Received unknown progress update, %v, of type %T, for support bundle ID: %s", val, val, bundle.ID)
				}
			case <-updateTicker.C:
				if err := store.GetStore().UpdateSupportBundle(bundle); err != nil {
					logger.Error(errors.Wrap(err, "could not update bundle"))
				}
			case <-timeout:
				logger.Errorf("Timeout exceeded for support bundle ID: %s", bundle.ID)
				if err := store.GetStore().UpdateSupportBundle(bundle); err != nil {
					logger.Error(errors.Wrap(err, "could not write failure to bundle"))
					return
				}
				return
			}
		}
	}()

	return progressChan
}

func typeof(msg interface{}) {
	panic("unimplemented")
}

// executeSupportBundleCollectRoutine creates a goroutine to collect the support bundle, upload, analyze and
// send redactors. The function takes a channel for progress updates and closes it when collectors are complete.
func executeSupportBundleCollectRoutine(bundle *types.SupportBundle, progressChan chan interface{}) {

	collectorCB := func(c chan interface{}, msg string) {
		c <- supportBundleProgressUpdate{
			Message: msg,
			Status:  types.BUNDLE_RUNNING,
			Type:    BUNDLE_PROGRESS_COLLECTOR,
		}
	}

	k8sconfig, err := k8sutil.GetClusterConfig()
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("Could not get k8s rest config for support bundle ID: %s", bundle.ID))
		logger.Error(err)
		return
	}

	opts := troubleshootv1beta2.SupportBundleCreateOpts{
		CollectorProgressCallback: collectorCB,
		CollectWithoutPermissions: true,
		HttpClient:                http.DefaultClient,
		KubernetesRestConfig:      k8sconfig,
		Namespace:                 "",
		ProgressChan:              progressChan,
		Redact:                    true,
		RunHostCollectorsInPod:    true, // always run host collectors in pod from KOTS regardless of the spec value
	}

	logger.Infof("Executing Collection go routine for support bundle ID: %s", bundle.ID)
	logger.Infof("Always run host collectors in pod: %t", opts.RunHostCollectorsInPod)

	go func() {
		defer close(progressChan)

		// redactions are global in troubleshoot....
		redact.ResetRedactionList()

		var response *troubleshootv1beta2.SupportBundleResponse
		if bundle.URI != "" {
			response, err = troubleshootv1beta2.CollectSupportBundleFromURI(bundle.URI, bundle.RedactURIs, opts)
			if err != nil {
				logger.Error(errors.Wrap(err, fmt.Sprintf("error collecting support bundle ID from URI: %s", bundle.ID)))
				return
			}
		} else if bundle.BundleSpec != nil {
			response, err = troubleshootv1beta2.CollectSupportBundleFromSpec(&bundle.BundleSpec.Spec, bundle.AdditionalRedactors, opts)
			if err != nil {
				logger.Error(errors.Wrap(err, fmt.Sprintf("error collecting support bundle ID: %s from spec", bundle.ID)))
				return
			}
		} else {
			logger.Errorf("cannot collect support bundle; no bundle URI or spec provided")
			return
		}
		defer os.RemoveAll(response.ArchivePath)

		progressChan <- supportBundleProgressUpdate{
			Message: "creating file tree",
			Status:  types.BUNDLE_RUNNING,
			Type:    BUNDLE_PROGRESS_FILETREE,
		}

		fileTree, err := archiveToFileTree(response.ArchivePath)
		if err != nil {
			progressChan <- supportBundleProgressUpdate{
				Message: "could not create file tree",
				Status:  types.BUNDLE_FAILED,
				Type:    BUNDLE_PROGRESS_ERROR,
			}
			logger.Error(errors.Wrap(err, "error parsing archive for tree"))
			return
		}

		marshalledTree, err := json.Marshal(fileTree.Nodes)
		if err != nil {
			progressChan <- supportBundleProgressUpdate{
				Message: "could not marshal tree",
				Status:  types.BUNDLE_FAILED,
				Type:    BUNDLE_PROGRESS_ERROR,
			}
			logger.Error(errors.Wrap(err, "error marshallling archive tree"))
			return
		}

		progressChan <- supportBundleProgressUpdate{
			Message: "uploading bundle to store",
			Status:  types.BUNDLE_RUNNING,
			Type:    BUNDLE_PROGRESS_UPLOADED,
		}

		if err = store.GetStore().UploadSupportBundle(bundle.ID, response.ArchivePath, marshalledTree); err != nil {
			progressChan <- supportBundleProgressUpdate{
				Message: "could not upload bundle",
				Status:  types.BUNDLE_FAILED,
				Type:    BUNDLE_PROGRESS_ERROR,
			}
			logger.Error(errors.Wrap(err, "error uploading the support bundle"))
			return
		}

		fi, err := os.Stat(response.ArchivePath)
		if err != nil {
			progressChan <- supportBundleProgressUpdate{
				Message: "could not get archive info",
				Status:  types.BUNDLE_FAILED,
				Type:    BUNDLE_PROGRESS_ERROR,
			}
			logger.Error(errors.Wrap(err, "error getting archive info"))
			return
		}

		size := float64(fi.Size())

		// last update is uploaded for parity with existing support bundles
		progressChan <- supportBundleProgressUpdate{
			Message: "support bundle uploaded",
			Status:  types.BUNDLE_UPLOADED,
			Type:    BUNDLE_PROGRESS_UPLOADED,
			Size:    &size,
		}

		// we need the app archive to get the analyzers for old support bundles that don't include the analysis in the bundle
		if err := CreateSupportBundleAnalysis(bundle.AppID, response.ArchivePath, bundle); err != nil {
			logger.Error(errors.Wrap(err, "failed to create analysis"))
			return
		}

		redactions := redact.GetRedactionList()
		if err = store.GetStore().SetRedactions(bundle.ID, redactions); err != nil {
			logger.Error(errors.Wrap(err, "failed to set redactions"))
			return
		}
	}()
}

func bundleErrorUpdate(bundle *types.SupportBundle) {
	logger.Errorf("Support bundle collection exited before completion for support bundle ID: %s", bundle.ID)
	bundle.Status = types.BUNDLE_FAILED

	if err := store.GetStore().UpdateSupportBundle(bundle); err != nil {
		logger.Error(errors.Wrap(err, "could not write failure to bundle"))
		return
	}
}
