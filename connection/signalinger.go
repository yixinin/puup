package connection

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/proto"
	"github.com/yixinin/puup/stderr"
)

type Signalinger interface {
	SendSdp(ctx context.Context, sdp webrtc.SessionDescription) error
	SendCandidate(ctx context.Context, ice *webrtc.ICECandidate) error
	RemoteSdp() chan webrtc.SessionDescription
	RemoteIceCandidates() chan webrtc.ICECandidate
}
type SigConfig struct {
	ServerAddr  string
	BackendName string
	FrontendKey string
}

type SignalingClient struct {
	Type        PeerType
	ServerAddr  string
	BackendName string
	FrontendKey string
}

func NewAnswerClient(serverAddr, backName string) *SignalingClient {
	c := newSignalingClient(Answer, serverAddr, backName)
	return c
}
func NewOfferClient(serverAddr, backName string) *SignalingClient {
	c := newSignalingClient(Offer, serverAddr, backName)
	c.FrontendKey = uuid.NewString()
	return c
}

func newSignalingClient(t PeerType, serverAddr, backName string) *SignalingClient {
	return &SignalingClient{
		Type:        t,
		ServerAddr:  serverAddr,
		BackendName: backName,
	}
}

func (c *SignalingClient) Clone() *SignalingClient {
	var cc = *c
	return &cc
}

func (c *SignalingClient) Keepalive(ctx context.Context, newConnEvent chan string) error {
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
		newConnEvent <- key
	}
	return nil
}

func (c *SignalingClient) GetConnectionInfo(ctx context.Context) (proto.GetConnectionInfoAck, error) {
	var ack proto.GetConnectionInfoAck
	resp, err := http.DefaultClient.Get(c.GetConnectionInfoURL())
	if err != nil {
		return ack, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&ack)
	return ack, err
}

func (c *SignalingClient) PostCandidate(ctx context.Context, cd *webrtc.ICECandidate) error {
	data, err := json.Marshal(proto.PostCandidateReq{
		Name:      c.BackendName,
		Key:       c.FrontendKey,
		Candidate: cd,
	})
	if err != nil {
		return stderr.Wrap(err)
	}
	_, err = http.DefaultClient.Post(c.GetCandidateURL(), "application/json", bytes.NewBuffer(data))
	return stderr.Wrap(err)
}

func (c *SignalingClient) PostSdp(ctx context.Context, sdp []byte) error {
	data, err := json.Marshal(proto.PostSdpReq{
		Name: c.BackendName,
		Key:  c.FrontendKey,
		Sdp:  sdp,
	})
	if err != nil {
		return stderr.Wrap(err)
	}
	_, err = http.DefaultClient.Post(c.GetSdpURL(), "application/json", bytes.NewBuffer(data))
	return stderr.Wrap(err)
}

func (c *SignalingClient) Offline(ctx context.Context) {
	var vals = url.Values{}
	vals.Add("name", c.BackendName)
	vals.Add("key", c.FrontendKey)
	http.Head(fmt.Sprintf("%s/api/offline?%s", c.ServerAddr, vals.Encode()))
}

func (c *SignalingClient) GetCandidateURL() string {
	return fmt.Sprintf("%s/api/candidate/%s?name=%s", c.ServerAddr, c.Type.Url(), c.BackendName)
}

func (c *SignalingClient) GetSdpURL() string {
	return fmt.Sprintf("%s/api/sdp/%s", c.ServerAddr, c.Type.Url())
}

func (c *SignalingClient) GetConnectionInfoURL() string {
	var vals = url.Values{}
	vals.Add("name", c.BackendName)
	vals.Add("key", c.FrontendKey)
	return fmt.Sprintf("%s/api/conninfo/%s?%s", c.ServerAddr, c.Type.Url(), vals.Encode())
}

func (c *SignalingClient) GetKeepAliveURL() string {
	return fmt.Sprintf("%s/api/keepalive/%s?name=%s", c.ServerAddr, c.Type.Url(), c.BackendName)
}
