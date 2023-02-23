package conn

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/proto"
	"github.com/yixinin/puup/stderr"
)

type Signalinger interface {
	NewClient() chan string
	SendSdp(ctx context.Context, id string, sdp webrtc.SessionDescription) error
	SendCandidate(ctx context.Context, id string, ty webrtc.SDPType, ice *webrtc.ICECandidate) error
	RemoteSdp(id string) chan webrtc.SessionDescription
	RemoteIceCandidates(id string) chan *webrtc.ICECandidate
	Offline(ctx context.Context, clientId string) error
}

type SignalingClient struct {
	sync.Mutex
	localType webrtc.SDPType

	sigAddr    string
	serverName string
	clientId   string

	newClient chan string
	sdps      map[string]chan webrtc.SessionDescription
	ices      map[string]chan *webrtc.ICECandidate
	close     chan struct{}
}

func NewSignalingClient(t webrtc.SDPType, sigAddr, serverName string) *SignalingClient {
	c := &SignalingClient{
		localType:  t,
		sigAddr:    sigAddr,
		serverName: serverName,
		newClient:  make(chan string, 1),
		sdps:       make(map[string]chan webrtc.SessionDescription),
		ices:       make(map[string]chan *webrtc.ICECandidate),
		close:      make(chan struct{}),
	}
	go c.loop()
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

func (c *SignalingClient) OnSdp(id string, sdp *webrtc.SessionDescription) {
	if sdp == nil {
		return
	}
	if c.localType == webrtc.SDPTypeAnswer {
		c.newClient <- id
	}
	c.RemoteSdp(id) <- *sdp
}
func (c *SignalingClient) OnCandidate(id string, ice *webrtc.ICECandidate) {
	c.GetIceChan(id) <- ice
}

func (c *SignalingClient) FetchSdp(tp webrtc.SDPType, id string) error {
	return c.Fetch(proto.GetFetchURL(c.sigAddr, tp, c.serverName, id))
}

func (c *SignalingClient) Fetch(url string) error {
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		logrus.Errorf("send keepalive error:%v", err)
		return stderr.Wrap(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 203 {
		return nil
	}
	if resp.StatusCode != 200 {
		logrus.Errorf("send keepalive resp %d != 200", resp.StatusCode)
		return stderr.Wrap(err)

	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("read keepalive resp error:%v", err)
		return stderr.Wrap(err)
	}
	var ack proto.FetchAck
	err = json.Unmarshal(data, &ack)
	if err != nil {
		logrus.Errorf("decode keepalive resp error:%v", err)
		return stderr.Wrap(err)
	}

	if ack.Sdp != nil {
		c.OnSdp(ack.Id, ack.Sdp)
	}
	for _, ice := range ack.Candidates {
		c.OnCandidate(ack.Id, ice)
	}
	return nil
}

func (c *SignalingClient) loop() {
	tk := time.NewTicker(200 * time.Millisecond)
	defer tk.Stop()
	for {
		select {
		case <-c.close:
			return
		case <-tk.C:
			err := c.FetchSdp(c.localType, c.clientId)
			if err != nil {
				logrus.Errorf("fetch error:%v", err)
			}
		}
	}
}

func (c *SignalingClient) SendCandidate(ctx context.Context, id string, tp webrtc.SDPType, ice *webrtc.ICECandidate) error {
	data, err := json.Marshal(proto.PostCandidateReq{
		Name:      c.serverName,
		Id:        id,
		Type:      tp,
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
	if err != nil {
		return stderr.Wrap(err)
	}
	c.clientId = id
	return nil
}

func (c *SignalingClient) Offline(ctx context.Context, clientId string) error {
	c.clientId = ""
	url := proto.GetOfflineURL(c.sigAddr, c.serverName, clientId)
	_, err := http.Head(url)
	return err
}
