package types

import (
	"time"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	Conn         *websocket.Conn `json:"-"`
	ConnectedAt  time.Time       `json:"connectedAt"`
	LastPingSent PingPongInfo    `json:"lastPingSent"`
	LastPongRecv PingPongInfo    `json:"lastPongRecv"`
	LastPingRecv PingPongInfo    `json:"lastPingRecv"`
	LastPongSent PingPongInfo    `json:"lastPongSent"`
}

type PingPongInfo struct {
	Time    time.Time `json:"time"`
	Message string    `json:"message"`
}
