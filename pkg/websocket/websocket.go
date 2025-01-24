package websocket

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/websocket/types"
)

var wsUpgrader = websocket.Upgrader{}

type ConnectionManager struct {
	clients map[string]types.WSClient
	mu      sync.Mutex
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		clients: make(map[string]types.WSClient),
	}
}

func (m *ConnectionManager) Connect(w http.ResponseWriter, r *http.Request, nodeName string, version string) error {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return errors.Wrap(err, "failed to upgrade to websocket")
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Time{})
	conn.SetWriteDeadline(time.Time{})
	conn.SetPingHandler(m.wsPingHandler(nodeName, version, conn))
	conn.SetPongHandler(m.wsPongHandler(nodeName, version))
	conn.SetCloseHandler(m.wsCloseHandler(nodeName, version))

	// register the client
	m.registerWSClient(nodeName, version, conn)

	// ping client on a regular interval to make sure it's still connected
	go m.pingWSClient(nodeName, version, conn)

	// listen to client messages
	m.listenToWSClient(nodeName, version, conn)
	return nil
}

func (m *ConnectionManager) GetClients() map[string]types.WSClient {
	m.mu.Lock()
	defer m.mu.Unlock()

	copy := make(map[string]types.WSClient)
	for k, v := range m.clients {
		copy[k] = v
	}
	return copy
}

// This is only used for testing
func (m *ConnectionManager) ResetClients() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, e := range m.clients {
		if e.Conn != nil {
			fmt.Println("reset close")
			e.Conn.Close()
		}
	}

	m.clients = make(map[string]types.WSClient)
}

func (m *ConnectionManager) pingWSClient(nodeName string, version string, conn *websocket.Conn) {
	for {
		sleepDuration := time.Second * time.Duration(5+rand.Intn(16)) // 5-20 seconds
		time.Sleep(sleepDuration)

		done := func() bool {
			m.mu.Lock()
			defer m.mu.Unlock()

			if m.clientChanged(nodeName, version) {
				return true
			}

			pingMsg := fmt.Sprintf("%x", rand.Int())
			if err := conn.WriteControl(websocket.PingMessage, []byte(pingMsg), time.Now().Add(1*time.Second)); err != nil {
				if m.isWSConnClosed(nodeName, version, err) {
					fmt.Println("PING ERROR", err)
					m.handleWSConnClosed(nodeName, version, err)
					return true
				}
				logger.Debugf("Failed to send ping message to %s: %v", nodeName, err)
				return false
			}

			client := m.clients[nodeName]
			client.LastPingSent = types.PingPongInfo{
				Time:    time.Now(),
				Message: pingMsg,
			}
			m.clients[nodeName] = client
			return false
		}()
		if done {
			break
		}
	}
}

func (m *ConnectionManager) listenToWSClient(nodeName string, version string, conn *websocket.Conn) {
	defer func() {
		fmt.Println("listenToWSClient done", nodeName)
	}()
	for {
		_, _, err := conn.ReadMessage() // this is required to receive ping/pong messages
		if err != nil {
			m.mu.Lock()
			if m.isWSConnClosed(nodeName, version, err) {
				m.handleWSConnClosed(nodeName, version, err)
				m.mu.Unlock()
				return
			}
			m.mu.Unlock()
			logger.Debugf("Error reading websocket message from %s: %v", nodeName, err)
		}
	}
}

func (m *ConnectionManager) registerWSClient(nodeName string, version string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e, ok := m.clients[nodeName]; ok && e.Conn != nil {
		fmt.Println("registerWSClient close existing connection", nodeName)
		e.Conn.Close()
	}

	m.clients[nodeName] = types.WSClient{
		Conn:        conn,
		ConnectedAt: time.Now(),
		Version:     version,
	}

	logger.Infof("Registered new websocket for %s with version %s", nodeName, version)
}

func (m *ConnectionManager) wsPingHandler(nodeName string, version string, conn *websocket.Conn) func(message string) error {
	return func(message string) error {
		m.mu.Lock()
		defer m.mu.Unlock()

		if m.clientChanged(nodeName, version) {
			fmt.Println("PING CLIENT CHANGED", nodeName, version)
			return nil
		}

		client := m.clients[nodeName]
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

		m.clients[nodeName] = client
		return nil
	}
}

func (m *ConnectionManager) wsPongHandler(nodeName string, version string) func(message string) error {
	return func(message string) error {
		m.mu.Lock()
		defer m.mu.Unlock()

		if m.clientChanged(nodeName, version) {
			fmt.Println("PONG CLIENT CHANGED", nodeName, version)
			return nil
		}

		client := m.clients[nodeName]
		client.LastPongRecv = types.PingPongInfo{
			Time:    time.Now(),
			Message: message,
		}
		m.clients[nodeName] = client

		return nil
	}
}

func (m *ConnectionManager) wsCloseHandler(nodeName string, version string) func(code int, text string) error {
	return func(code int, text string) error {
		m.mu.Lock()
		defer m.mu.Unlock()

		m.handleWSConnClosed(nodeName, version, errors.Errorf("%d (exit code), message: %q", code, text))
		return nil
	}
}

func (m *ConnectionManager) isWSConnClosed(nodeName string, version string, err error) bool {
	if m.clientChanged(nodeName, version) {
		fmt.Println("isWSConnClosed CLIENT CHANGED", nodeName, version)
		return true
	}
	if _, ok := err.(*websocket.CloseError); ok {
		return true
	}
	if e, ok := err.(*net.OpError); ok && !e.Temporary() {
		return true
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}

func (m *ConnectionManager) handleWSConnClosed(nodeName string, version string, err error) {
	logger.Infof("Websocket connection closed for %s with version %s: %v", nodeName, version, err)

	// do not delete if the client has changed
	if m.clientChanged(nodeName, version) {
		return
	}

	delete(m.clients, nodeName)
}

func (m *ConnectionManager) clientChanged(nodeName string, version string) bool {
	if e, ok := m.clients[nodeName]; !ok || e.Version != version {
		return true
	}
	return false
}
