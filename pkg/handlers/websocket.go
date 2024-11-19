package handlers

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"golang.org/x/exp/rand"
)

var wsUpgrader = websocket.Upgrader{}
var wsClients = make(map[string]*websocket.Conn)
var wsMutex = sync.Mutex{}

type ConnectToECWebsocketResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) ConnectToECWebsocket(w http.ResponseWriter, r *http.Request) {
	response := ConnectToECWebsocketResponse{
		Success: false,
	}

	clientID := r.URL.Query().Get("id")
	if clientID == "" {
		response.Error = "missing client id"
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

	conn.SetPingHandler(wsPingHandler(clientID, conn))
	conn.SetPongHandler(wsPongHandler(clientID, conn))
	conn.SetCloseHandler(wsCloseHandler(clientID, conn))

	// register the client
	registerWSClient(clientID, conn)

	// ping client on a regular interval to make sure it's still connected
	go pingWSClient(clientID, conn)

	// listen to client messages
	listenToWSClient(clientID, conn)
}

func pingWSClient(id string, conn *websocket.Conn) {
	for {
		sleepDuration := time.Second * time.Duration(5+rand.Intn(16)) // 5-20 seconds
		time.Sleep(sleepDuration)

		pingMsg := fmt.Sprintf("%d", rand.Intn(1000))
		logger.Infof("Sending ping message '%s' to client %s", pingMsg, id)

		if err := conn.WriteControl(websocket.PingMessage, []byte(pingMsg), time.Now().Add(1*time.Second)); err != nil {
			if isWSConnClosed(id, err) {
				return
			}
			logger.Errorf("Failed to send ping message to client %s: %v", id, err)
		}
	}
}

func listenToWSClient(id string, conn *websocket.Conn) {
	for {
		_, _, err := conn.ReadMessage() // this is required to receive ping/pong messages
		if err != nil {
			if isWSConnClosed(id, err) {
				return
			}
			logger.Errorf("Error reading websocket message from client %s: %v", id, err)
		}
	}
}

func registerWSClient(id string, conn *websocket.Conn) {
	wsMutex.Lock()
	defer wsMutex.Unlock()

	if existingConn, ok := wsClients[id]; ok {
		existingConn.Close()
		delete(wsClients, id)
	}
	wsClients[id] = conn

	logger.Infof("Registered new websocket client %s", id)
}

func wsPingHandler(id string, conn *websocket.Conn) func(message string) error {
	return func(message string) error {
		logger.Infof("Received ping message '%s' from client %s", message, id)
		logger.Infof("Sending pong message '%s' to client %s", message, id)
		if err := conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(1*time.Second)); err != nil {
			logger.Errorf("Failed to send pong message to client %s: %v", id, err)
		}
		return nil
	}
}

func wsPongHandler(id string, conn *websocket.Conn) func(message string) error {
	return func(message string) error {
		logger.Infof("Received pong message '%s' from client %s", message, id)
		return nil
	}
}

func wsCloseHandler(id string, conn *websocket.Conn) func(code int, text string) error {
	return func(code int, text string) error {
		logger.Errorf("Websocket connection closed for client %s: %d (exit code), message: %q", id, code, text)

		wsMutex.Lock()
		delete(wsClients, id)
		wsMutex.Unlock()

		closeMessage := websocket.FormatCloseMessage(code, text)
		if err := conn.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(time.Second)); err != nil {
			logger.Errorf("Failed to send close message to client %s: %v", id, err)
		}
		return nil
	}
}

func isWSConnClosed(id string, err error) bool {
	if _, ok := wsClients[id]; !ok {
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
