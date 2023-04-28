package net

import (
	"context"
	"net"
	"sync"

	"github.com/google/uuid"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/proto"
)

type PeersClient struct {
	sync.Mutex
	cluster map[string]PeerClient
}

func NewPeersClient() *PeersClient {
	return &PeersClient{
		cluster: make(map[string]PeerClient),
	}
}

func (c *PeersClient) addPeer(serverName string, p *conn.Peer) {
	c.Lock()
	defer c.Unlock()
	if m, ok := c.cluster[serverName]; ok {
		m.peers[p.Id] = p
	} else {
		c.cluster[serverName] = PeerClient{
			clusterName: serverName,
			peers: map[string]*conn.Peer{
				p.Id: p,
			},
		}
	}
}

func (c *PeersClient) getPeer(serverName, clientId string) (*conn.Peer, bool) {
	c.Lock()
	defer c.Unlock()
	m, ok := c.cluster[serverName]
	if ok && m.peers != nil {
		if p, ok := m.peers[clientId]; ok {
			return p, true
		}
	}
	return nil, false
}

func (c *PeersClient) getRandPeer(serverName string) (*conn.Peer, bool) {
	c.Lock()
	defer c.Unlock()
	if m, ok := c.cluster[serverName]; ok {
		for _, v := range m.peers {
			return v, true
		}
	}

	return nil, false
}

func (c *PeersClient) GetCluster() ([]string, error) {
	var ack proto.GetClusterAck

	return ack.Ids, nil
}
func (c *PeersClient) GetCluserClient(wsURL, clusterName string) PeerClient {
	c.Lock()
	defer c.Unlock()
	cc, ok := c.cluster[clusterName]
	if !ok {
		cc = PeerClient{
			sig:         conn.NewWsFrontendSigClient(uuid.NewString(), wsURL, clusterName),
			peers:       make(map[string]*conn.Peer),
			clusterName: clusterName,
		}
		c.cluster[clusterName] = cc
	}
	return cc
}
func (c *PeersClient) Connect(wsURL, clusterName string) error {
	cids, err := c.GetCluster()
	if err != nil {
		return err
	}
	cc := c.GetCluserClient(wsURL, clusterName)
	for _, cid := range cids {
		peer, err := conn.NewOfferPeer(cc.sig, cid)
		if err != nil {
			return err
		}
		if err = peer.Connect(context.TODO()); err != nil {
			return err
		}
		c.addPeer(clusterName, peer)
	}

	return nil
}

func (c *PeersClient) Dial(wsURL string, clusterName string, ct conn.ChannelType) (net.Conn, error) {
	_, ok := c.getRandPeer(clusterName)
	if !ok {
		var err error
		err = c.Connect(wsURL, clusterName)
		if err != nil {
			return nil, err
		}
	}
	for _, p := range c.cluster[clusterName].peers {
		cc, err := p.Get(ct)
		if err != nil {
			return nil, err
		}
		conn := NewConn(cc)
		return conn, nil
	}
	return nil, nil
}
