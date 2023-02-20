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

	peers map[string]*conn.Peer
}

func NewPeersClient() *PeersClient {
	return &PeersClient{
		peers: make(map[string]*conn.Peer),
	}
}

func (c *PeersClient) addPeer(backName string, p *conn.Peer) {
	c.Lock()
	defer c.Unlock()
	c.peers[backName] = p
}

func (c *PeersClient) getPeer(serverName string) (*conn.Peer, bool) {
	c.Lock()
	defer c.Unlock()
	p, ok := c.peers[serverName]
	if ok && p != nil && !p.IsClose() {
		return p, true
	}
	if ok {
		delete(c.peers, serverName)
	}
	return nil, false
}

func (c *PeersClient) Connect(sigAddr, serverName string) error {
	_, ok := c.getPeer(serverName)
	if ok {
		return nil
	}
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return stderr.Wrap(err)
	}
	sigCli := conn.NewSignalingClient(conn.Offer, sigAddr, serverName)
	peer, err := conn.NewOfferPeer(pc, sigCli)
	if err != nil {
		return err
	}
	if err = peer.Connect(context.TODO()); err != nil {
		return err
	}
	c.addPeer(serverName, peer)
	return nil
}

func (c *PeersClient) DialWeb(sigAddr, serverName string) (net.Conn, error) {
	p, ok := c.getPeer(serverName)
	if ok {
		c, err := p.GetWebConn()
		if err != nil {
			return nil, err
		}
		conn := NewConn(c, p.ReleaseChan())
		return conn, nil
	}
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return nil, stderr.Wrap(err)
	}
	sigCli := conn.NewSignalingClient(conn.Offer, sigAddr, serverName)
	p, err = conn.NewOfferPeer(pc, sigCli)
	if err != nil {
		return nil, err
	}
	if err = p.Connect(context.TODO()); err != nil {
		return nil, err
	}
	c.addPeer(serverName, p)

	cc, err := p.GetWebConn()
	if err != nil {
		return nil, err
	}
	conn := NewConn(cc, p.ReleaseChan())
	return conn, nil
}

func (c *PeersClient) DialFile(sigAddr, serverName string) (net.Conn, error) {
	p, ok := c.getPeer(serverName)
	if ok {
		cc, err := p.GetFileConn()
		if err != nil {
			return nil, err
		}
		conn := NewConn(cc, p.ReleaseChan())
		return conn, nil
	}
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return nil, stderr.Wrap(err)
	}
	sig := conn.NewSignalingClient(conn.Offer, sigAddr, serverName)
	p, err = conn.NewOfferPeer(pc, sig)
	if err != nil {
		return nil, err
	}
	if err = p.Connect(context.TODO()); err != nil {
		return nil, err
	}
	c.addPeer(serverName, p)

	cc, err := p.GetFileConn()
	if err != nil {
		return nil, err
	}
	conn := NewConn(cc, p.ReleaseChan())
	return conn, nil
}

func (c *PeersClient) DialSsh(sigAddr, serverName string) (net.Conn, error) {
	p, ok := c.getPeer(serverName)
	if ok {
		cc, err := p.GetSshConn()
		if err != nil {
			return nil, err
		}
		conn := NewConn(cc, p.ReleaseChan())
		return conn, nil
	}
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return nil, stderr.Wrap(err)
	}
	sig := conn.NewSignalingClient(conn.Offer, sigAddr, serverName)
	p, err = conn.NewOfferPeer(pc, sig)
	if err != nil {
		return nil, err
	}
	if err = p.Connect(context.TODO()); err != nil {
		return nil, err
	}
	c.addPeer(serverName, p)

	cc, err := p.GetSshConn()
	if err != nil {
		return nil, err
	}
	conn := NewConn(cc, p.ReleaseChan())
	return conn, nil
}

func (c *PeersClient) DialProxy(sigAddr, serverName string, port uint16) (net.Conn, error) {
	p, ok := c.getPeer(serverName)
	if ok {
		cc, err := p.GetProxyConn(port)
		if err != nil {
			return nil, err
		}
		conn := NewConn(cc, p.ReleaseChan())
		return conn, nil
	}
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return nil, stderr.Wrap(err)
	}
	sig := conn.NewSignalingClient(conn.Offer, sigAddr, serverName)
	p, err = conn.NewOfferPeer(pc, sig)
	if err != nil {
		return nil, err
	}
	if err = p.Connect(context.TODO()); err != nil {
		return nil, err
	}
	c.addPeer(serverName, p)

	cc, err := p.GetProxyConn(port)
	if err != nil {
		return nil, err
	}
	conn := NewConn(cc, p.ReleaseChan())
	return conn, nil
}
