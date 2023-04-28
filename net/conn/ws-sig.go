package conn

import (
	"context"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/proto"
)

type WsSigClient struct {
	sync.RWMutex
	cancel context.CancelFunc
	websocket.Upgrader
	wsURL string
	conn  *websocket.Conn

	id          string
	clusterName string
	Type        webrtc.SDPType

	sessions map[string]*Session

	OnSession func(id, cid string)
	isClose   bool
}

type Session struct {
	sync.RWMutex
	id     string
	closed bool
	sdp    chan webrtc.SessionDescription
	ice    chan *webrtc.ICECandidate
}

func NewSession(id string) *Session {
	return &Session{
		id:  id,
		sdp: make(chan webrtc.SessionDescription, 1),
		ice: make(chan *webrtc.ICECandidate, 5),
	}
}

func (c *WsSigClient) Id() string {
	return c.id
}
func (s *Session) Close() {
	if s == nil {
		return
	}
	s.Lock()
	defer s.Unlock()
	s.closed = true
	close(s.ice)
	close(s.sdp)
}
func (s *Session) IsClose() bool {
	if s == nil {
		return true
	}
	s.RLock()
	defer s.RUnlock()
	return s.closed
}

func NewWsSigClient(id, wsURL, clusterName string) *WsSigClient {
	return &WsSigClient{
		wsURL:       wsURL,
		isClose:     true,
		id:          id,
		clusterName: clusterName,
		sessions:    make(map[string]*Session, 1),
	}
}

func (c *WsSigClient) CloseSession(id string) {
	c.RLock()
	defer c.RUnlock()
	sess, ok := c.sessions[id]
	if !ok {
		return

	}
	sess.Close()
	time.AfterFunc(5*time.Second, func() {
		c.Lock()
		defer c.Unlock()
		delete(c.sessions, id)
	})
}

func (c *WsSigClient) GetSession(id string) *Session {
	c.Lock()
	defer c.Unlock()
	sess, ok := c.sessions[id]
	if ok {
		return sess
	}
	if c.OnSession != nil {
		c.OnSession(id, cid)
	}
	sess = NewSession(id)
	c.sessions[id] = sess
	return sess
}

func (s *Session) OnSdp(sdp *webrtc.SessionDescription) {
	if sdp == nil || s == nil {
		return
	}

	if s.IsClose() {
		return
	}

	s.sdp <- *sdp
}

func (s *Session) OnIceCandidate(ice *webrtc.ICECandidate) {
	if ice == nil || s == nil {
		return
	}

	if s.IsClose() {
		return
	}

	s.ice <- ice
}

func (c *WsSigClient) SendPacket(ctx context.Context, p proto.Packet) error {
	return c.conn.WriteJSON(p)
}
func (c *WsSigClient) RemoteSdp(id string) chan webrtc.SessionDescription {
	return c.GetSession(id).sdp
}
func (c *WsSigClient) RemoteIceCandidates(id string) chan *webrtc.ICECandidate {
	return c.GetSession(id).ice
}

func (c *WsSigClient) Run(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.Dial(c.wsURL, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	defer func() {
		if r := recover(); r != nil {
			logrus.WithField("stacks", string(debug.Stack())).Errorf("sig client paniced:%v", err)
		}
		c.isClose = true
		conn.Close()
		cancel()
	}()

	var header = proto.WsHeader{
		Type: c.Type,
		Id:   c.id,
		Name: c.clusterName,
	}
	if err := conn.WriteJSON(header); err != nil {
		return err
	}

	c.isClose = false
loop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var packet proto.Packet
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			err := conn.ReadJSON(&packet)
			if os.IsTimeout(err) {
				continue loop
			}
			if err != nil {
				return err
			}

			if packet.To.PeerId == "" {
				continue loop
			}
			sess := c.GetSession(packet.To.PeerId)
			if sess.IsClose() {
				continue loop
			}
			sess.OnIceCandidate(packet.ICECandidate)
			sess.OnSdp(packet.Sdp)
		}
	}
}

func (c *WsSigClient) Close(ctx context.Context) error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

func (c *WsSigClient) IsClose() bool {
	return c.isClose
}
