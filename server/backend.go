package server

import (
	"net"

	"github.com/gorilla/websocket"
)

const NoCluster = "nc"

type Client struct {
	Id    string
	Peers map[string]string
	conn  *websocket.Conn
}

func (c *Client) Close() {
	if c == nil || c.conn == nil {
		return
	}
	c.conn.Close()
}
func (c *Client) Send(v any) error {
	if c.conn == nil || v == nil {
		return net.ErrClosed
	}
	return c.conn.WriteJSON(v)
}
