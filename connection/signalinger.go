package connection

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/proto"
	"github.com/yixinin/puup/stderr"
)

type Signalinger interface {
	NewClient() chan string
	SendSdp(ctx context.Context, id string, sdp webrtc.SessionDescription) error
	SendCandidate(ctx context.Context, id string, ice *webrtc.ICECandidate) error
	RemoteSdp(id string) chan webrtc.SessionDescription
	RemoteIceCandidates(id string) chan webrtc.ICECandidate
}

type SignalingClient struct {
	Type        PeerType
	ServerAddr  string
	BackendName string

	newClient chan string
	sdps      map[string]chan webrtc.SessionDescription
	ices      map[string]chan webrtc.ICECandidate
	close     chan struct{}
}

func NewSignalingClient(t PeerType, serverAddr, backName string) *SignalingClient {
	c := &SignalingClient{
		Type:        t,
		ServerAddr:  serverAddr,
		BackendName: backName,
		newClient:   make(chan string, 1),
		sdps:        make(map[string]chan webrtc.SessionDescription),
		ices:        make(map[string]chan webrtc.ICECandidate),
		close:       make(chan struct{}),
	}
	go c.loop()
	return c
}

func (c *SignalingClient) keepalive() error {
	resp, err := http.DefaultClient.Get(c.GetKeepAliveURL())
	if err != nil {
		logrus.Errorf("send keepalive error:%v", err)
		return stderr.Wrap(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		logrus.Errorf("send keepalive resp %d != 200", resp.StatusCode)
		stderr.Wrap(err)
	}
	var ack proto.KeepaliveAck
	err = json.NewDecoder(resp.Body).Decode(&ack)
	if err != nil {
		logrus.Errorf("decode keepalive resp error:%v", err)
		stderr.Wrap(err)
	}
	for _, key := range ack.Keys {

		c.newClient <- key
	}
	return nil
}

func (c *SignalingClient) loop() {
	for {
		select {
		case <-c.close:
			return
		default:
			err := c.keepalive()
			if err != nil {
				logrus.Errorf("keep alive error:%v", err)
			}
		}
	}
}

func (c *SignalingClient) GetConnectionInfo(ctx context.Context, id string) (proto.GetConnectionInfoAck, error) {
	var ack proto.GetConnectionInfoAck
	resp, err := http.DefaultClient.Get(c.GetConnectionInfoURL(id))
	if err != nil {
		return ack, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&ack)
	return ack, err
}

func (c *SignalingClient) SendCandidate(ctx context.Context, id string, ice *webrtc.ICECandidate) error {
	data, err := json.Marshal(proto.PostCandidateReq{
		Name:      c.BackendName,
		Key:       id,
		Candidate: ice,
	})
	if err != nil {
		return stderr.Wrap(err)
	}
	_, err = http.DefaultClient.Post(c.GetCandidateURL(), "application/json", bytes.NewBuffer(data))
	return stderr.Wrap(err)
}

func (c *SignalingClient) SendSdp(ctx context.Context, id string, sdp webrtc.SessionDescription) error {
	data, err := json.Marshal(proto.PostSdpReq{
		Name: c.BackendName,
		Key:  id,
		Sdp:  sdp,
	})
	if err != nil {
		return stderr.Wrap(err)
	}
	_, err = http.DefaultClient.Post(c.GetSdpURL(), "application/json", bytes.NewBuffer(data))
	return stderr.Wrap(err)
}

func (c *SignalingClient) Offline(ctx context.Context, id string) {
	var vals = url.Values{}
	vals.Add("name", c.BackendName)
	vals.Add("key", id)
	http.Head(fmt.Sprintf("%s/api/offline?%s", c.ServerAddr, vals.Encode()))
}

func (c *SignalingClient) GetCandidateURL() string {
	return fmt.Sprintf("%s/api/candidate/%s?name=%s", c.ServerAddr, c.Type.Url(), c.BackendName)
}

func (c *SignalingClient) GetSdpURL() string {
	return fmt.Sprintf("%s/api/sdp/%s", c.ServerAddr, c.Type.Url())
}

func (c *SignalingClient) GetConnectionInfoURL(id string) string {
	var vals = url.Values{}
	vals.Add("name", c.BackendName)
	vals.Add("key", id)
	return fmt.Sprintf("%s/api/conninfo/%s?%s", c.ServerAddr, c.Type.Url(), vals.Encode())
}

func (c *SignalingClient) GetKeepAliveURL() string {
	return fmt.Sprintf("%s/api/keepalive/%s?name=%s", c.ServerAddr, c.Type.Url(), c.BackendName)
}
