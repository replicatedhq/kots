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

// UpgradeCluster sends an upgrade command to the first available websocket from the active ones
func UpgradeCluster(installation *ecv1beta1.Installation, appSlug, versionLabel, stepID string) error {
	marshalledInst, err := json.Marshal(installation)
	if err != nil {
		return errors.Wrap(err, "marshal installation")
	}

	data, err := json.Marshal(map[string]string{
		"installation": string(marshalledInst),
		"appSlug":      appSlug,
		"versionLabel": versionLabel,
		"stepID":       stepID,
	})
	if err != nil {
		return errors.Wrap(err, "marshal installation")
	}

	message, err := json.Marshal(map[string]interface{}{
		"command": "upgrade-cluster",
		"data":    string(data),
	})
	if err != nil {
		return errors.Wrap(err, "marshal command message")
	}

	wscli, nodeName, err := firstActiveWSClient()
	if err != nil {
		return errors.Wrap(err, "get first active websocket client")
	}

	logger.Infof("Sending cluster upgrade command to websocket of node %s with message: %s", nodeName, string(message))

	if err := wscli.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return errors.Wrap(err, "send upgrade command to websocket")
	}

	return nil
}

func AddExtension(repos []k0sv1beta1.Repository, chart ecv1beta1.Chart, appSlug, versionLabel, stepID string) error {
	return sendExtensionCommand("add-extension", repos, chart, appSlug, versionLabel, stepID)
}

func UpgradeExtension(repos []k0sv1beta1.Repository, chart ecv1beta1.Chart, appSlug, versionLabel, stepID string) error {
	return sendExtensionCommand("upgrade-extension", repos, chart, appSlug, versionLabel, stepID)
}

func RemoveExtension(repos []k0sv1beta1.Repository, chart ecv1beta1.Chart, appSlug, versionLabel, stepID string) error {
	return sendExtensionCommand("remove-extension", repos, chart, appSlug, versionLabel, stepID)
}

func sendExtensionCommand(command string, repos []k0sv1beta1.Repository, chart ecv1beta1.Chart, appSlug, versionLabel, stepID string) error {
	marshalledRepos, err := json.Marshal(repos)
	if err != nil {
		return errors.Wrap(err, "marshal repos")
	}

	marshalledChart, err := json.Marshal(chart)
	if err != nil {
		return errors.Wrap(err, "marshal chart")
	}

	data, err := json.Marshal(map[string]string{
		"repos":        string(marshalledRepos),
		"chart":        string(marshalledChart),
		"appSlug":      appSlug,
		"versionLabel": versionLabel,
		"stepID":       stepID,
	})
	if err != nil {
		return errors.Wrap(err, "marshal data")
	}

	message, err := json.Marshal(map[string]interface{}{
		"command": command,
		"data":    string(data),
	})
	if err != nil {
		return errors.Wrap(err, "marshal command message")
	}

	wscli, nodeName, err := firstActiveWSClient()
	if err != nil {
		return errors.Wrap(err, "get first active websocket client")
	}

	logger.Infof("Sending extension %s command to websocket of node %s with message: %s", command, nodeName, string(message))

	if err := wscli.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return errors.Wrap(err, "send upgrade command to websocket")
	}

	return nil
}

func firstActiveWSClient() (types.WSClient, string, error) {
	wsMutex.Lock()
	defer wsMutex.Unlock()

	if len(wsClients) == 0 {
		return types.WSClient{}, "", errors.New("no active websocket connections available")
	}

	var wscli types.WSClient
	var nodeName string

	// get the first client in the map
	for name, client := range wsClients {
		nodeName = name
		wscli = client
		break
	}

	return wscli, nodeName, nil
}
