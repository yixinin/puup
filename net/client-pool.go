package net

import (
	"sync"

	"github.com/google/uuid"
	"github.com/yixinin/puup/net/conn"
)

type PeerClient struct {
	sync.RWMutex
	clusterName string
	peers       map[string]*conn.Peer
	sig         conn.Signalinger
}

func NewPeerClient(wsURL, clusterName string) (*PeerClient, error) {
	p := &PeerClient{
		clusterName: clusterName,
		peers:       make(map[string]*conn.Peer),
		sig:         conn.NewWsFrontendSigClient(uuid.NewString(), wsURL, clusterName),
	}

	return p, nil
}
