package connection

import (
	"net"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/ice"
	"github.com/yixinin/puup/stderr"
)

type PeersClient struct {
	sync.Mutex

	peers map[string]*Peer
}

func NewPeersClient() *PeersClient {
	return &PeersClient{
		peers: make(map[string]*Peer),
	}
}

func (c *PeersClient) addPeer(backName string, p *Peer) {
	c.Lock()
	defer c.Unlock()
	c.peers[backName] = p
}

func (c *PeersClient) getPeer(backendName string) (*Peer, bool) {
	c.Lock()
	defer c.Unlock()
	p, ok := c.peers[backendName]
	if ok && p != nil && !p.IsClose() {
		return p, true
	}
	if ok {
		delete(c.peers, backendName)
	}
	return nil, false
}

func (c *PeersClient) Connect(serverAddr, backendName string) error {
	p, ok := c.getPeer(backendName)
	if ok {
		return nil
	}
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return stderr.Wrap(err)
	}
	p, err = NewOfferPeer(pc, NewOfferClient(serverAddr, backendName))
	if err != nil {
		return err
	}
	if err = p.Connect(); err != nil {
		return err
	}
	c.addPeer(backendName, p)
	return nil
}

func (c *PeersClient) Dial(serverAddr, backendName string) (net.Conn, error) {
	p, ok := c.getPeer(backendName)
	if ok {
		return p.GetWebConn("")
	}
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return nil, stderr.Wrap(err)
	}
	p, err = NewOfferPeer(pc, NewOfferClient(serverAddr, backendName))
	if err != nil {
		return nil, err
	}
	if err = p.Connect(); err != nil {
		return nil, err
	}
	c.addPeer(backendName, p)

	return p.GetWebConn("")
}

func (c *PeersClient) DialProxy(serverAddr, backendName string, port uint16) (net.Conn, error) {
	p, ok := c.getPeer(backendName)
	if ok {
		return p.GetProxyConn(port)
	}
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return nil, stderr.Wrap(err)
	}
	p, err = NewOfferPeer(pc, NewOfferClient(serverAddr, backendName))
	if err != nil {
		return nil, err
	}
	if err = p.Connect(); err != nil {
		return nil, err
	}
	c.addPeer(backendName, p)

	return p.GetProxyConn(port)
}
