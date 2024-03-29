package conn

import (
	"context"

	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/proto"
)

type ClientPeer struct {
	ClientId string
	PeerId   string
}
type Signalinger interface {
	Id() string
	NewPeer() chan ClientPeer
	SendPacket(ctx context.Context, p proto.Packet) error
	RemoteSdp(id string) chan webrtc.SessionDescription
	RemoteIceCandidates(id string) chan *webrtc.ICECandidate
	Run(ctx context.Context) error
	Close(ctx context.Context) error
	CloseSession(id string)
	IsClose() bool
}

type SigStatus string

const (
	StatusIdle      SigStatus = "idle"
	StatusFetch     SigStatus = "fetch"
	StatusKeepalive SigStatus = "kl"
)

// type SignalingClient struct {
// 	sync.Mutex
// 	localType webrtc.SDPType

// 	num atomic.Int32

// 	sigAddr    string
// 	serverName string
// 	clientId   string

// 	newClient chan string
// 	sdps      map[string]chan webrtc.SessionDescription
// 	ices      map[string]chan *webrtc.ICECandidate
// 	close     chan struct{}
// }

// func NewSignalingClient(t webrtc.SDPType, sigAddr, serverName string) *SignalingClient {
// 	c := &SignalingClient{
// 		localType:  t,
// 		sigAddr:    sigAddr,
// 		serverName: serverName,
// 		newClient:  make(chan string, 1),
// 		sdps:       make(map[string]chan webrtc.SessionDescription),
// 		ices:       make(map[string]chan *webrtc.ICECandidate),
// 		close:      make(chan struct{}),
// 	}
// 	go c.loop()
// 	return c
// }

// func (c *SignalingClient) Status() int32 {
// 	return c.num.Load()
// }

// func (c *SignalingClient) Start() int32 {
// 	return c.num.Add(1)
// }
// func (c *SignalingClient) End() int32 {
// 	return c.num.Add(-1)
// }
// func (c *SignalingClient) Pause() {
// 	c.num.Store(-1)
// }

// func (c *SignalingClient) NewClient() chan string {
// 	return c.newClient
// }
// func (c *SignalingClient) RemoteIceCandidates(id string) chan *webrtc.ICECandidate {
// 	c.Lock()
// 	defer c.Unlock()
// 	ch, ok := c.ices[id]
// 	if !ok {
// 		ch = make(chan *webrtc.ICECandidate)
// 		c.ices[id] = ch
// 	}
// 	return ch

// }
// func (c *SignalingClient) RemoteSdp(id string) chan webrtc.SessionDescription {
// 	c.Lock()
// 	defer c.Unlock()

// 	ch, ok := c.sdps[id]
// 	if !ok {
// 		ch = make(chan webrtc.SessionDescription)
// 		c.sdps[id] = ch
// 	}
// 	return ch
// }

// func (c *SignalingClient) GetIceChan(id string) chan *webrtc.ICECandidate {
// 	c.Lock()
// 	defer c.Unlock()

// 	ch, ok := c.ices[id]
// 	if !ok {
// 		ch = make(chan *webrtc.ICECandidate)
// 		c.ices[id] = ch
// 	}
// 	return ch
// }

// func (c *SignalingClient) OnSdp(id string, sdp *webrtc.SessionDescription) {
// 	if sdp == nil {
// 		return
// 	}
// 	if c.localType == webrtc.SDPTypeAnswer {
// 		c.newClient <- id
// 	}
// 	c.RemoteSdp(id) <- *sdp
// }
// func (c *SignalingClient) OnCandidate(id string, ice *webrtc.ICECandidate) {
// 	c.GetIceChan(id) <- ice
// }

// func (c *SignalingClient) Fetch(url string) error {
// 	resp, err := http.DefaultClient.Get(url)
// 	if err != nil {
// 		logrus.Errorf("send keepalive error:%v", err)
// 		return stderr.Wrap(err)
// 	}
// 	defer resp.Body.Close()
// 	if resp.StatusCode == 203 {
// 		return nil
// 	}
// 	if resp.StatusCode != 200 {
// 		logrus.Errorf("send keepalive resp %d != 200", resp.StatusCode)
// 		return stderr.Wrap(err)

// 	}
// 	data, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		logrus.Errorf("read keepalive resp error:%v", err)
// 		return stderr.Wrap(err)
// 	}
// 	var ack proto.FetchAck
// 	err = json.Unmarshal(data, &ack)
// 	if err != nil {
// 		logrus.Errorf("decode keepalive resp error:%v", err)
// 		return stderr.Wrap(err)
// 	}

// 	if ack.Sdp != nil {
// 		c.OnSdp(ack.Id, ack.Sdp)
// 	}
// 	for _, ice := range ack.Candidates {
// 		c.OnCandidate(ack.Id, ice)
// 	}
// 	return nil
// }

// func (c *SignalingClient) loop() {
// 	tk := time.NewTicker(200 * time.Millisecond)
// 	defer tk.Stop()
// 	var i uint8
// 	for {
// 		i++
// 		select {
// 		case <-c.close:
// 			return
// 		case <-tk.C:
// 			switch c.Status() {
// 			case -1:
// 			case 1:
// 				err := c.Fetch(proto.GetFetchURL(c.sigAddr, c.localType, c.serverName, c.clientId))
// 				if err != nil {
// 					logrus.Errorf("fetch error:%v", err)
// 				}
// 			default:
// 				if i%16 == 0 {
// 					err := c.Fetch(proto.GetFetchURL(c.sigAddr, c.localType, c.serverName, c.clientId))
// 					if err != nil {
// 						logrus.Errorf("fetch error:%v", err)
// 					}
// 				}
// 			}
// 		}
// 	}
// }

// func (c *SignalingClient) SendCandidate(ctx context.Context, id string, tp webrtc.SDPType, ice *webrtc.ICECandidate) error {
// 	data, err := json.Marshal(proto.PostCandidateReq{
// 		Name:      c.serverName,
// 		Id:        id,
// 		Type:      tp,
// 		Candidate: ice,
// 	})
// 	if err != nil {
// 		return stderr.Wrap(err)
// 	}
// 	_, err = http.DefaultClient.Post(proto.GetPostCandidateURL(c.sigAddr), "application/json", bytes.NewBuffer(data))
// 	return stderr.Wrap(err)
// }

// func (c *SignalingClient) SendSdp(ctx context.Context, id string, sdp webrtc.SessionDescription) error {

// 	data, err := json.Marshal(proto.PostSdpReq{
// 		Name: c.serverName,
// 		Id:   id,
// 		Sdp:  sdp,
// 	})
// 	if err != nil {
// 		return stderr.Wrap(err)
// 	}
// 	_, err = http.DefaultClient.Post(proto.GetPostSdpURL(c.sigAddr), "application/json", bytes.NewBuffer(data))
// 	if err != nil {
// 		return stderr.Wrap(err)
// 	}
// 	c.clientId = id
// 	return nil
// }

// func (c *SignalingClient) Offline(ctx context.Context, clientId string) error {
// 	c.clientId = ""
// 	url := proto.GetOfflineURL(c.sigAddr, c.serverName, clientId)
// 	_, err := http.Head(url)
// 	return err
// }
