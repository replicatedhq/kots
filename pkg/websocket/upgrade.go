package websocket

import (
	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	ecv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/websocket/types"
)

// UpgradeCluster sends an upgrade command to the first available websocket from the active ones
func UpgradeCluster(installation *ecv1beta1.Installation, appSlug, versionLabel, stepID string) error {
	wsMutex.Lock()
	defer wsMutex.Unlock()

	if len(wsClients) == 0 {
		return errors.New("no active websocket connections available")
	}

	var selectedClient types.WSClient
	var nodeName string

	// get the first client in the map
	for name, client := range wsClients {
		nodeName = name
		selectedClient = client
		break
	}

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

	logger.Infof("Sending cluster upgrade command to websocket of node %s with message: %s", nodeName, string(message))

	if err := selectedClient.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return errors.Wrap(err, "send upgrade command to websocket")
	}

	return nil
}
