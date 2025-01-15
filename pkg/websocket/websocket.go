package websocket

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/websocket/types"
)

var wsUpgrader = websocket.Upgrader{}
var wsClients = make(map[string]types.WSClient)
var wsMutex = sync.Mutex{}

func Connect(w http.ResponseWriter, r *http.Request, nodeName string, version string) error {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return errors.Wrap(err, "failed to upgrade to websocket")
	}
	defer conn.Close()

	conn.SetPingHandler(wsPingHandler(nodeName, conn))
	conn.SetPongHandler(wsPongHandler(nodeName))
	conn.SetCloseHandler(wsCloseHandler(nodeName, version))

	// register the client
	registerWSClient(nodeName, version, conn)

	// ping client on a regular interval to make sure it's still connected
	go pingWSClient(nodeName, version, conn)

	// listen to client messages
	listenToWSClient(nodeName, version, conn)
	return nil
}

func pingWSClient(nodeName string, version string, conn *websocket.Conn) {
	for {
		sleepDuration := time.Second * time.Duration(5+rand.Intn(16)) // 5-20 seconds
		time.Sleep(sleepDuration)

		pingMsg := fmt.Sprintf("%x", rand.Int())

		if err := conn.WriteControl(websocket.PingMessage, []byte(pingMsg), time.Now().Add(1*time.Second)); err != nil {
			if isWSConnClosed(nodeName, err) {
				handleWSConnClosed(nodeName, version, err)
				return
			}
			logger.Debugf("Failed to send ping message to %s: %v", nodeName, err)
			continue
		}

		wsMutex.Lock()
		client := wsClients[nodeName]
		client.LastPingSent = types.PingPongInfo{
			Time:    time.Now(),
			Message: pingMsg,
		}
		wsClients[nodeName] = client
		wsMutex.Unlock()
	}
}

func listenToWSClient(nodeName string, version string, conn *websocket.Conn) {
	for {
		_, _, err := conn.ReadMessage() // this is required to receive ping/pong messages
		if err != nil {
			if isWSConnClosed(nodeName, err) {
				handleWSConnClosed(nodeName, version, err)
				return
			}
			logger.Debugf("Error reading websocket message from %s: %v", nodeName, err)
		}
	}
}

func registerWSClient(nodeName string, version string, conn *websocket.Conn) {
	wsMutex.Lock()
	defer wsMutex.Unlock()

	if e, ok := wsClients[nodeName]; ok {
		e.Conn.Close()
		delete(wsClients, nodeName)
	}

	wsClients[nodeName] = types.WSClient{
		Conn:        conn,
		ConnectedAt: time.Now(),
		Version:     version,
	}

	logger.Infof("Registered new websocket for %s with version %s", nodeName, version)
}

func wsPingHandler(nodeName string, conn *websocket.Conn) func(message string) error {
	return func(message string) error {
		wsMutex.Lock()
		defer wsMutex.Unlock()

		client := wsClients[nodeName]
		client.LastPingRecv = types.PingPongInfo{
			Time:    time.Now(),
			Message: message,
		}

		if err := conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(1*time.Second)); err != nil {
			logger.Debugf("Failed to send pong message to %s: %v", nodeName, err)
		} else {
			client.LastPongSent = types.PingPongInfo{
				Time:    time.Now(),
				Message: message,
			}
		}

		wsClients[nodeName] = client
		return nil
	}
}

func wsPongHandler(nodeName string) func(message string) error {
	return func(message string) error {
		wsMutex.Lock()
		defer wsMutex.Unlock()

		client := wsClients[nodeName]
		client.LastPongRecv = types.PingPongInfo{
			Time:    time.Now(),
			Message: message,
		}
		wsClients[nodeName] = client

		return nil
	}
}

func wsCloseHandler(nodeName string, version string) func(code int, text string) error {
	return func(code int, text string) error {
		handleWSConnClosed(nodeName, version, errors.Errorf("%d (exit code), message: %q", code, text))
		return nil
	}
}

func isWSConnClosed(nodeName string, err error) bool {
	wsMutex.Lock()
	defer wsMutex.Unlock()

	if _, ok := wsClients[nodeName]; !ok {
		return true
	}
	if _, ok := err.(*websocket.CloseError); ok {
		return true
	}
	if e, ok := err.(*net.OpError); ok && !e.Temporary() {
		return true
	}
	return false
}

func handleWSConnClosed(nodeName string, version string, err error) {
	wsMutex.Lock()
	defer wsMutex.Unlock()

	logger.Infof("Websocket connection closed for %s with version %s: %v", nodeName, version, err)

	// if the version is different, it means a new version of
	// the client has connected and we should ignore
	if e, ok := wsClients[nodeName]; ok && e.Version != version {
		return
	}

	delete(wsClients, nodeName)
}

func GetClients() map[string]types.WSClient {
	return wsClients
}
