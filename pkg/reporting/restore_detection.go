package reporting

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/segmentio/ksuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// environmentFingerprint identifies the environment a kotsadm installation is running in.
// It is stored in the kotsadm database so that it is included in snapshots; after a
// restore, the stored fingerprint describes the environment the backup was taken in.
// See RESTORE_DETECTION.md for the full design.
type environmentFingerprint struct {
	KubeSystemUID   string `json:"kubeSystemUID,omitempty"`
	PodNamespaceUID string `json:"podNamespaceUID,omitempty"`
}

type fingerprintDecision int

const (
	decisionKeep fingerprintDecision = iota
	decisionRegenerate
)

// compareEnvironmentFingerprints decides whether the installation is still running in the
// environment recorded in the stored fingerprint. The kube-system namespace UID is the
// authoritative cluster identity; the pod namespace UID is only consulted when kube-system
// is not readable (namespace-scoped RBAC). When no field is comparable the instance ID is
// kept: regenerating on uncertainty would fragment reporting for healthy installations.
func compareEnvironmentFingerprints(stored environmentFingerprint, current environmentFingerprint) fingerprintDecision {
	if stored.KubeSystemUID != "" && current.KubeSystemUID != "" {
		if stored.KubeSystemUID == current.KubeSystemUID {
			return decisionKeep
		}
		return decisionRegenerate
	}

	if stored.PodNamespaceUID != "" && current.PodNamespaceUID != "" {
		if stored.PodNamespaceUID == current.PodNamespaceUID {
			return decisionKeep
		}
		return decisionRegenerate
	}

	return decisionKeep
}

func getCurrentEnvironmentFingerprint(clientset kubernetes.Interface) environmentFingerprint {
	fingerprint := environmentFingerprint{}

	// this runs synchronously during startup; bound it so an unresponsive API server
	// cannot block kotsadm from coming up
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if ns, err := clientset.CoreV1().Namespaces().Get(ctx, metav1.NamespaceSystem, metav1.GetOptions{}); err == nil {
		fingerprint.KubeSystemUID = string(ns.UID)
	} else {
		logger.Debugf("failed to get kube-system namespace for environment fingerprint: %v", err)
	}

	if util.PodNamespace != "" {
		if ns, err := clientset.CoreV1().Namespaces().Get(ctx, util.PodNamespace, metav1.GetOptions{}); err == nil {
			fingerprint.PodNamespaceUID = string(ns.UID)
		} else {
			logger.Debugf("failed to get pod namespace for environment fingerprint: %v", err)
		}
	}

	return fingerprint
}

// resolveAppInstanceID returns the instance ID to report for the app and the instance ID
// it was restored from (empty if the app was never restored into a new environment).
// Falls back to the app ID so that reporting keeps working if the store is unavailable.
func resolveAppInstanceID(kotsStore store.Store, appID string) (instanceID string, restoredFrom string) {
	instanceID, lineage, err := kotsStore.GetAppInstanceID(appID)
	if err != nil {
		logger.Warnf("failed to get instance id for app %s, falling back to app id: %v", appID, err)
		return appID, ""
	}

	if len(lineage) > 0 {
		restoredFrom = lineage[len(lineage)-1]
	}

	return instanceID, restoredFrom
}

// checkForEnvironmentRestore detects whether the kotsadm database was restored from a
// snapshot taken in a different environment, and if so generates a new instance ID for
// each installed app, recording the previous ID in the app's restore lineage.
func checkForEnvironmentRestore(clientset kubernetes.Interface, kotsStore store.Store) error {
	current := getCurrentEnvironmentFingerprint(clientset)
	if current == (environmentFingerprint{}) {
		logger.Infof("skipping snapshot restore detection: environment fingerprint is not readable")
		return nil
	}

	currentJSON, err := json.Marshal(current)
	if err != nil {
		return errors.Wrap(err, "failed to marshal environment fingerprint")
	}

	storedJSON, err := kotsStore.GetEnvironmentFingerprint()
	if err != nil {
		return errors.Wrap(err, "failed to get stored environment fingerprint")
	}

	if storedJSON == "" {
		// first boot with restore detection (fresh install or upgrade): adopt the current
		// environment without touching instance IDs
		return errors.Wrap(kotsStore.SetEnvironmentFingerprint(string(currentJSON)), "failed to adopt environment fingerprint")
	}

	stored := environmentFingerprint{}
	if err := json.Unmarshal([]byte(storedJSON), &stored); err != nil {
		logger.Infof("stored environment fingerprint is not parseable, re-adopting current environment: %v", err)
		return errors.Wrap(kotsStore.SetEnvironmentFingerprint(string(currentJSON)), "failed to re-adopt environment fingerprint")
	}

	if compareEnvironmentFingerprints(stored, current) == decisionKeep {
		// refresh with merged values: a transiently unreadable field (e.g. an RBAC or
		// API hiccup on kube-system) must not weaken the stored fingerprint, or a later
		// same-cluster DR would misclassify as a restore into a new environment
		merged := current
		if merged.KubeSystemUID == "" {
			merged.KubeSystemUID = stored.KubeSystemUID
		}
		if merged.PodNamespaceUID == "" {
			merged.PodNamespaceUID = stored.PodNamespaceUID
		}
		mergedJSON, err := json.Marshal(merged)
		if err != nil {
			return errors.Wrap(err, "failed to marshal merged environment fingerprint")
		}
		if storedJSON != string(mergedJSON) {
			return errors.Wrap(kotsStore.SetEnvironmentFingerprint(string(mergedJSON)), "failed to refresh environment fingerprint")
		}
		return nil
	}

	apps, err := kotsStore.ListInstalledApps()
	if err != nil {
		return errors.Wrap(err, "failed to list installed apps")
	}

	for _, app := range apps {
		previousInstanceID, lineage, err := kotsStore.GetAppInstanceID(app.ID)
		if err != nil {
			return errors.Wrapf(err, "failed to get instance id for app %s", app.ID)
		}

		newInstanceID := ksuid.New().String()
		lineage = append(lineage, previousInstanceID)

		if err := kotsStore.SetAppInstanceID(app.ID, newInstanceID, lineage); err != nil {
			return errors.Wrapf(err, "failed to set instance id for app %s", app.ID)
		}

		logger.Infof("detected restore into a different environment: generated new instance id %s for app %s (restored from %s)", newInstanceID, app.ID, previousInstanceID)
	}

	// persist the fingerprint last so that a failure above re-runs detection on next startup
	return errors.Wrap(kotsStore.SetEnvironmentFingerprint(string(currentJSON)), "failed to store environment fingerprint")
}
