package net

import (
	"context"
	"net"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/ice"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/stderr"
)

type PeersClient struct {
	sync.Mutex
	peers map[string]map[string]*conn.Peer
}

func NewPeersClient() *PeersClient {
	return &PeersClient{
		peers: make(map[string]map[string]*conn.Peer),
	}
}

func (c *PeersClient) addPeer(serverName, clientId string, p *conn.Peer) {
	c.Lock()
	defer c.Unlock()
	if m, ok := c.peers[serverName]; ok {
		m[clientId] = p
	} else {
		m := make(map[string]*conn.Peer, 1)
		m[clientId] = p
		c.peers[serverName] = m
	}
}

func (c *PeersClient) getPeer(serverName, clientId string) (*conn.Peer, bool) {
	c.Lock()
	defer c.Unlock()
	m, ok := c.peers[serverName]
	if ok && m != nil {
		if p, ok := m[clientId]; ok {
			return p, true
		}
	}
	return nil, false
}

func (c *PeersClient) getRandPeer(serverName string) (*conn.Peer, bool) {
	c.Lock()
	defer c.Unlock()
	if m, ok := c.peers[serverName]; ok {
		for _, v := range m {
			return v, true
		}
	}

	return nil, false
}

func (c *PeersClient) Connect(sigAddr, serverName string) (*conn.Peer, error) {
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return nil, stderr.Wrap(err)
	}
	sigCli := conn.NewSignalingClient(webrtc.SDPTypeOffer, sigAddr, serverName)
	sigCli.Pause()
	peer, err := conn.NewOfferPeer(pc, serverName, sigCli)
	if err != nil {
		return nil, err
	}
	if err = peer.Connect(context.TODO()); err != nil {
		return nil, err
	}
	c.addPeer(serverName, peer.ClientId(), peer)
	return peer, nil
}

func (c *PeersClient) Dial(sigAddr, serverName string, ct conn.ChannelType) (net.Conn, error) {
	p, ok := c.getRandPeer(serverName)
	if !ok {
		var err error
		p, err = c.Connect(sigAddr, serverName)
		if err != nil {
			return nil, err
		}
	}

	cc, err := p.Get(ct)
	if err != nil {
		return nil, err
	}
	conn := NewConn(cc)
	return conn, nil
}
