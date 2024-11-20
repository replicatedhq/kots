package embeddedcluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/logger"
)

// UpgradeStartedEvent is send back home when the upgrade starts.
type UpgradeStartedEvent struct {
	ClusterID      string `json:"clusterID"`
	TargetVersion  string `json:"targetVersion"`
	InitialVersion string `json:"initialVersion"`
}

// UpgradeFailedEvent is send back home when the upgrade fails.
type UpgradeFailedEvent struct {
	ClusterID      string `json:"clusterID"`
	TargetVersion  string `json:"targetVersion"`
	InitialVersion string `json:"initialVersion"`
	Reason         string `json:"reason"`
}

// UpgradeSucceededEvent event is send back home when the upgrade succeeds.
type UpgradeSucceededEvent struct {
	ClusterID      string `json:"clusterID"`
	TargetVersion  string `json:"targetVersion"`
	InitialVersion string `json:"initialVersion"`
}

// NotifyUpgradeStarted notifies the metrics server that an upgrade has started.
func NotifyUpgradeStarted(ctx context.Context, baseURL string, ins, prev *embeddedclusterv1beta1.Installation) error {
	if ins.Spec.AirGap {
		return nil
	}
	return sendEvent(ctx, "UpgradeStarted", baseURL, UpgradeStartedEvent{
		ClusterID:      ins.Spec.ClusterID,
		TargetVersion:  ins.Spec.Config.Version,
		InitialVersion: prev.Spec.Config.Version,
	})
}

// NotifyUpgradeFailed notifies the metrics server that an upgrade has failed.
func NotifyUpgradeFailed(ctx context.Context, baseURL string, ins, prev *embeddedclusterv1beta1.Installation, reason string) error {
	if ins.Spec.AirGap {
		return nil
	}
	return sendEvent(ctx, "UpgradeFailed", baseURL, UpgradeFailedEvent{
		ClusterID:      ins.Spec.ClusterID,
		TargetVersion:  ins.Spec.Config.Version,
		InitialVersion: prev.Spec.Config.Version,
		Reason:         reason,
	})
}

// NotifyUpgradeSucceeded notifies the metrics server that an upgrade has succeeded.
func NotifyUpgradeSucceeded(ctx context.Context, baseURL string, ins, prev *embeddedclusterv1beta1.Installation) error {
	if ins.Spec.AirGap {
		return nil
	}
	return sendEvent(ctx, "UpgradeSucceeded", baseURL, UpgradeSucceededEvent{
		ClusterID:      ins.Spec.ClusterID,
		TargetVersion:  ins.Spec.Config.Version,
		InitialVersion: prev.Spec.Config.Version,
	})
}

// sendEvent sends the received event to the metrics server through a post request.
func sendEvent(ctx context.Context, evname, baseURL string, ev interface{}) error {
	url := fmt.Sprintf("%s/embedded_cluster_metrics/%s", baseURL, evname)

	logger.Infof("Sending event %s to %s", evname, url)

	body := map[string]interface{}{"event": ev}
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send event: %s", resp.Status)
	}
	return nil
}
