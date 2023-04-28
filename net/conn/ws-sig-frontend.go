package conn

import "github.com/pion/webrtc/v3"

type WsFrontendSigClient struct {
	*WsSigClient
}

func NewWsFrontendSigClient(id, wsURL, clusterName string) *WsFrontendSigClient {
	c := &WsFrontendSigClient{
		WsSigClient: NewWsSigClient(id, wsURL, clusterName),
	}

	c.Type = webrtc.SDPTypeOffer
	return c
}
func (c *WsFrontendSigClient) NewClient() chan string {
	return nil
}
