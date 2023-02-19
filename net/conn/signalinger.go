package conn

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"

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
	RemoteIceCandidates(id string) chan *webrtc.ICECandidate
	Offline(ctx context.Context, clientId string) error
}

type SignalingClient struct {
	sync.RWMutex
	localType PeerType

	sigAddr    string
	serverName string

	newClient chan string
	sdps      map[string]chan webrtc.SessionDescription
	ices      map[string]chan *webrtc.ICECandidate
	close     chan struct{}
}

func NewSignalingClient(t PeerType, serverAddr, backName string) *SignalingClient {
	c := &SignalingClient{
		localType:  t,
		sigAddr:    serverAddr,
		serverName: backName,
		newClient:  make(chan string, 1),
		sdps:       make(map[string]chan webrtc.SessionDescription),
		ices:       make(map[string]chan *webrtc.ICECandidate),
		close:      make(chan struct{}),
	}
	go c.loop()
	return c
}

func (c *SignalingClient) Clone(clientId string) *SignalingClient {
	c = NewSignalingClient(c.localType, c.sigAddr, c.serverName)

	return c
}

func (c *SignalingClient) NewClient() chan string {
	return c.newClient
}
func (c *SignalingClient) RemoteIceCandidates(id string) chan *webrtc.ICECandidate {
	c.Lock()
	defer c.Unlock()
	ch, ok := c.ices[id]
	if !ok {
		ch = make(chan *webrtc.ICECandidate)
		c.ices[id] = ch
	}
	return ch

}
func (c *SignalingClient) RemoteSdp(id string) chan webrtc.SessionDescription {
	c.Lock()
	defer c.Unlock()

	ch, ok := c.sdps[id]
	if !ok {
		ch = make(chan webrtc.SessionDescription)
		c.sdps[id] = ch
	}
	return ch
}

func (c *SignalingClient) GetIceChan(id string) chan *webrtc.ICECandidate {
	c.Lock()
	defer c.Unlock()

	ch, ok := c.ices[id]
	if !ok {
		ch = make(chan *webrtc.ICECandidate)
		c.ices[id] = ch
	}
	return ch
}

func (c *SignalingClient) OnSdp(id string, sdp webrtc.SessionDescription) {
	if c.localType == Answer {
		c.newClient <- id
	}
	c.RemoteSdp(id) <- sdp
}
func (c *SignalingClient) OnCandidate(id string, ice *webrtc.ICECandidate) {
	c.GetIceChan(id) <- ice
}

func (c *SignalingClient) FetchOffer() error {
	return c.Fetch(proto.GetFetchURL(c.sigAddr, c.serverName, ""))
}

func (c *SignalingClient) FetchAnswer(id string) error {
	return c.Fetch(proto.GetFetchURL(c.sigAddr, c.serverName, id))
}

func (c *SignalingClient) Fetch(url string) error {
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		logrus.Errorf("send keepalive error:%v", err)
		return stderr.Wrap(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		logrus.Errorf("send keepalive resp %d != 200", resp.StatusCode)
		return stderr.Wrap(err)

	}
	var ack proto.FetchAck
	err = json.NewDecoder(resp.Body).Decode(&ack)
	if err != nil {
		logrus.Errorf("decode keepalive resp error:%v", err)
		return stderr.Wrap(err)
	}

	if ack.Sdp.SDP != "" {
		c.OnSdp(ack.Id, ack.Sdp)
	}
	for _, ice := range ack.Candidates {
		c.OnCandidate(ack.Id, ice)
	}
	return nil
}

func (c *SignalingClient) loop() {
	for {
		select {
		case <-c.close:
			return
		default:
			err := c.FetchOffer()
			if err != nil {
				logrus.Errorf("fetch error:%v", err)
			}
		}
	}
}

func (c *SignalingClient) SendCandidate(ctx context.Context, id string, ice *webrtc.ICECandidate) error {
	data, err := json.Marshal(proto.PostCandidateReq{
		Name:      c.serverName,
		Id:        id,
		Candidate: ice,
	})
	if err != nil {
		return stderr.Wrap(err)
	}
	_, err = http.DefaultClient.Post(proto.GetPostCandidateURL(c.sigAddr), "application/json", bytes.NewBuffer(data))
	return stderr.Wrap(err)
}

func (c *SignalingClient) SendSdp(ctx context.Context, id string, sdp webrtc.SessionDescription) error {
	data, err := json.Marshal(proto.PostSdpReq{
		Name: c.serverName,
		Id:   id,
		Sdp:  sdp,
	})
	if err != nil {
		return stderr.Wrap(err)
	}
	_, err = http.DefaultClient.Post(proto.GetPostSdpURL(c.sigAddr), "application/json", bytes.NewBuffer(data))
	return stderr.Wrap(err)
}

func (c *SignalingClient) Offline(ctx context.Context, clientId string) error {
	url := proto.GetOfflineURL(c.sigAddr, c.serverName, clientId)
	_, err := http.Head(url)
	return err
}
