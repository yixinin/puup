package conn

import "github.com/pion/webrtc/v3"

type WsBackendSigClient struct {
	*WsSigClient

	newClient chan ClientPeer
}

func NewWsBackendSigClient(id, wsURL, clusterName string) *WsBackendSigClient {
	c := &WsBackendSigClient{
		WsSigClient: NewWsSigClient(id, wsURL, clusterName),
		newClient:   make(chan ClientPeer, 1),
	}
	c.WsSigClient.OnSession = c.OnSession
	c.WsSigClient.Type = webrtc.SDPTypeAnswer
	return c
}
func (c *WsBackendSigClient) NewClient() chan ClientPeer {
	return c.newClient
}
func (c *WsBackendSigClient) OnSession(id, clientId string) {
	c.newClient <- ClientPeer{
		ClientId: clientId,
		PeerId:   id,
	}
}
