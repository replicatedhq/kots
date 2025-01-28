package websocket

import (
	"encoding/json"

	"github.com/gorilla/websocket"
	k0sv1beta1 "github.com/k0sproject/k0s/pkg/apis/k0s/v1beta1"
	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/websocket/types"
)

// UpgradeECManager sends a manager upgrade command to all managers that are not running the specified version
func UpgradeECManager(m *ConnectionManager, nodeName, licenseID, licenseEndpoint, version, appSlug, versionLabel, stepID string) error {
	data, err := json.Marshal(types.UpgradeManagerData{
		LicenseID:       licenseID,
		LicenseEndpoint: licenseEndpoint,
	})
	if err != nil {
		return errors.Wrap(err, "marshal data")
	}

	message, err := json.Marshal(types.Message{
		AppSlug:      appSlug,
		VersionLabel: versionLabel,
		StepID:       stepID,
		Command:      types.CommandUpgradeManager,
		Data:         string(data),
	})
	if err != nil {
		return errors.Wrap(err, "marshal message")
	}

	wscli, err := wsClientForNode(m, nodeName)
	if err != nil {
		return errors.Wrapf(err, "get websocket client for node %s", nodeName)
	}

	if wscli.Version == version {
		logger.Infof("Embedded cluster manager on node %s is already running version %s. Skipping...", nodeName, version)
		return nil
	}

	logger.Infof("Sending ec manager upgrade command to websocket of node %s", nodeName)

	if err := wscli.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return errors.Wrap(err, "send upgrade ec manager command to websocket")
	}

	return nil
}

// UpgradeCluster sends an upgrade command to the first available websocket from the active ones
func UpgradeCluster(m *ConnectionManager, installation *ecv1beta1.Installation, appSlug, versionLabel, stepID string) error {
	data, err := json.Marshal(types.UpgradeClusterData{
		Installation: *installation,
	})
	if err != nil {
		return errors.Wrap(err, "marshal data")
	}

	message, err := json.Marshal(types.Message{
		AppSlug:      appSlug,
		VersionLabel: versionLabel,
		StepID:       stepID,
		Command:      types.CommandUpgradeCluster,
		Data:         string(data),
	})
	if err != nil {
		return errors.Wrap(err, "marshal message")
	}

	wscli, nodeName, err := firstActiveWSClient(m)
	if err != nil {
		return errors.Wrap(err, "get first active websocket client")
	}

	logger.Infof("Sending cluster upgrade command to websocket of node %s", nodeName)

	if err := wscli.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return errors.Wrap(err, "send upgrade command to websocket")
	}

	return nil
}

func AddExtension(m *ConnectionManager, repos []k0sv1beta1.Repository, chart ecv1beta1.Chart, appSlug, versionLabel, stepID string) error {
	return sendExtensionCommand(m, types.CommandAddExtension, repos, chart, appSlug, versionLabel, stepID)
}

func UpgradeExtension(m *ConnectionManager, repos []k0sv1beta1.Repository, chart ecv1beta1.Chart, appSlug, versionLabel, stepID string) error {
	return sendExtensionCommand(m, types.CommandUpgradeExtension, repos, chart, appSlug, versionLabel, stepID)
}

func RemoveExtension(m *ConnectionManager, repos []k0sv1beta1.Repository, chart ecv1beta1.Chart, appSlug, versionLabel, stepID string) error {
	return sendExtensionCommand(m, types.CommandRemoveExtension, repos, chart, appSlug, versionLabel, stepID)
}

func sendExtensionCommand(m *ConnectionManager, command types.Command, repos []k0sv1beta1.Repository, chart ecv1beta1.Chart, appSlug, versionLabel, stepID string) error {
	data, err := json.Marshal(types.ExtensionData{
		Repos: repos,
		Chart: chart,
	})
	if err != nil {
		return errors.Wrap(err, "marshal data")
	}

	message, err := json.Marshal(types.Message{
		AppSlug:      appSlug,
		VersionLabel: versionLabel,
		StepID:       stepID,
		Command:      command,
		Data:         string(data),
	})
	if err != nil {
		return errors.Wrap(err, "marshal message")
	}

	wscli, nodeName, err := firstActiveWSClient(m)
	if err != nil {
		return errors.Wrap(err, "get first active websocket client")
	}

	logger.Infof("Sending %s %s command to websocket of node %s", command, chart.Name, nodeName)

	if err := wscli.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return errors.Wrap(err, "send upgrade command to websocket")
	}

	return nil
}

func firstActiveWSClient(m *ConnectionManager) (types.WSClient, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.clients) == 0 {
		return types.WSClient{}, "", errors.New("no active websocket connections available")
	}

	var wscli types.WSClient
	var nodeName string

	// get the first client in the map
	for name, client := range m.clients {
		nodeName = name
		wscli = client
		break
	}

	return wscli, nodeName, nil
}

func wsClientForNode(m *ConnectionManager, nodeName string) (*types.WSClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, client := range m.clients {
		if name == nodeName {
			return &client, nil
		}
	}

	return nil, errors.New("not found")
}
