package server

import (
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

const NoCluster = "nc"

type Client struct {
	Id   string
	conn *websocket.Conn
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

type Cluster struct {
	sync.RWMutex
	Name      string
	backends  map[string]*Client
	frontends map[string]*Client
}

func NewCluster(name string) *Cluster {
	return &Cluster{
		Name:      name,
		backends:  make(map[string]*Client, 1),
		frontends: make(map[string]*Client, 1),
	}
}
func (c *Cluster) AddBackend(id string, conn *websocket.Conn) {
	c.Lock()
	defer c.Unlock()
	if v, ok := c.backends[id]; ok {
		v.Close()
	}
	c.backends[id] = &Client{id, conn}
}
func (c *Cluster) AddFrontend(id string, conn *websocket.Conn) {
	c.Lock()
	defer c.Unlock()
	if v, ok := c.frontends[id]; ok {
		v.Close()
	}
	c.frontends[id] = &Client{id, conn}
}

func (c *Cluster) DelBackend(id string) {
	c.Lock()
	defer c.Unlock()
	if v, ok := c.backends[id]; ok {
		v.Close()
		delete(c.backends, id)
	}
}

func (c *Cluster) DelFrontend(id string) {
	c.Lock()
	defer c.Unlock()
	if v, ok := c.frontends[id]; ok {
		v.Close()
		delete(c.frontends, id)
	}
}

func (c *Cluster) GetBackend(id string) (*Client, bool) {
	c.RLock()
	defer c.RUnlock()
	b, ok := c.backends[id]
	return b, ok
}

func (c *Cluster) GetFrontend(id string) (*Client, bool) {
	c.RLock()
	defer c.RUnlock()
	f, ok := c.frontends[id]
	return f, ok
}
