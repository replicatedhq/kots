package socket

import (
	"fmt"
	"strconv"

	"github.com/replicatedhq/kotsadm/operator/pkg/socket/transport"
)

const (
	webSocketProtocol       = "ws://"
	webSocketSecureProtocol = "wss://"
	socketioUrl             = "/socket.io/?EIO=3&transport=websocket&token=%s"
)

/**
Socket.io client representation
*/
type Client struct {
	methods
	Channel
}

/**
Get ws/wss url by host and port
*/
func GetUrl(host string, port int, auth string, secure bool) string {
	var prefix string
	if secure {
		prefix = webSocketSecureProtocol
	} else {
		prefix = webSocketProtocol
	}

	u := fmt.Sprintf(socketioUrl, auth)
	result := prefix + host + ":" + strconv.Itoa(port) + u

	return result
}

func NewClient() *Client {
	c := &Client{}
	c.initChannel()
	c.initMethods()
	return c
}

/**
connect to host and initialise socket.io protocol

The correct ws protocol url example:
ws://myserver.com/socket.io/?EIO=3&transport=websocket

You can use GetUrlByHost for generating correct url
*/
func (c *Client) Dial(url string, tr transport.Transport) error {
	var err error
	c.conn, err = tr.Connect(url)
	if err != nil {
		return err
	}

	// nspMsg := fmt.Sprintf("4%d%s", protocol.MessageTypeOpen, nsp)
	// c.conn.WriteMessage(nspMsg)

	go inLoop(&c.Channel, &c.methods)
	go outLoop(&c.Channel, &c.methods)
	go pinger(&c.Channel)

	return nil
}

/**
Close client connection
*/
func (c *Client) Close() {
	closeChannel(&c.Channel, &c.methods)
}
