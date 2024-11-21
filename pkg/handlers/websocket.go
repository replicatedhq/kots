package handlers

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
)

var wsUpgrader = websocket.Upgrader{}
var wsClients = make(map[string]*websocket.Conn)
var wsMutex = sync.Mutex{}

type ConnectToECWebsocketResponse struct {
	Error string `json:"error,omitempty"`
}

func (h *Handler) ConnectToECWebsocket(w http.ResponseWriter, r *http.Request) {
	response := ConnectToECWebsocketResponse{}

	nodeName := r.URL.Query().Get("nodeName")
	if nodeName == "" {
		response.Error = "missing node name"
		logger.Error(errors.New(response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		response.Error = "failed to upgrade to ws connection"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}
	defer conn.Close()

	conn.SetPingHandler(wsPingHandler(nodeName, conn))
	conn.SetPongHandler(wsPongHandler(nodeName, conn))
	conn.SetCloseHandler(wsCloseHandler(nodeName, conn))

	// register the client
	registerWSClient(nodeName, conn)

	// ping client on a regular interval to make sure it's still connected
	go pingWSClient(nodeName, conn)

	// listen to client messages
	listenToWSClient(nodeName, conn)
}

func pingWSClient(nodeName string, conn *websocket.Conn) {
	for {
		sleepDuration := time.Second * time.Duration(5+rand.Intn(16)) // 5-20 seconds
		time.Sleep(sleepDuration)

		pingMsg := fmt.Sprintf("%d", rand.Intn(1000))
		logger.Infof("Sending ping message '%s' to %s", pingMsg, nodeName)

		if err := conn.WriteControl(websocket.PingMessage, []byte(pingMsg), time.Now().Add(1*time.Second)); err != nil {
			if isWSConnClosed(nodeName, err) {
				return
			}
			logger.Errorf("Failed to send ping message to %s: %v", nodeName, err)
		}
	}
}

func listenToWSClient(nodeName string, conn *websocket.Conn) {
	for {
		_, _, err := conn.ReadMessage() // this is required to receive ping/pong messages
		if err != nil {
			if isWSConnClosed(nodeName, err) {
				return
			}
			logger.Errorf("Error reading websocket message from %s: %v", nodeName, err)
		}
	}
}

func registerWSClient(nodeName string, conn *websocket.Conn) {
	wsMutex.Lock()
	defer wsMutex.Unlock()

	if existingConn, ok := wsClients[nodeName]; ok {
		existingConn.Close()
		delete(wsClients, nodeName)
	}
	wsClients[nodeName] = conn

	logger.Infof("Registered new websocket for %s", nodeName)
}

func wsPingHandler(nodeName string, conn *websocket.Conn) func(message string) error {
	return func(message string) error {
		logger.Infof("Received ping message '%s' from %s", message, nodeName)
		logger.Infof("Sending pong message '%s' to %s", message, nodeName)
		if err := conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(1*time.Second)); err != nil {
			logger.Errorf("Failed to send pong message to %s: %v", nodeName, err)
		}
		return nil
	}
}

func wsPongHandler(nodeName string, conn *websocket.Conn) func(message string) error {
	return func(message string) error {
		logger.Infof("Received pong message '%s' from %s", message, nodeName)
		return nil
	}
}

func wsCloseHandler(nodeName string, conn *websocket.Conn) func(code int, text string) error {
	return func(code int, text string) error {
		logger.Errorf("Websocket connection closed for %s: %d (exit code), message: %q", nodeName, code, text)

		wsMutex.Lock()
		delete(wsClients, nodeName)
		wsMutex.Unlock()

		closeMessage := websocket.FormatCloseMessage(code, text)
		if err := conn.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(time.Second)); err != nil {
			logger.Errorf("Failed to send close message to %s: %v", nodeName, err)
		}
		return nil
	}
}

func isWSConnClosed(nodeName string, err error) bool {
	if _, ok := wsClients[nodeName]; !ok {
		return true
	}
	if _, ok := err.(*websocket.CloseError); ok {
		return true
	}
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	return false
}
