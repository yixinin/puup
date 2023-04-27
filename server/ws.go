package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

func (s *Server) WsSignalling(c *gin.Context) {
	conn, err := s.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.String(http.StatusBadRequest, "upgrade failed, error:%v", err)
	}
	go s.HandleWs(c.Request.Context(), conn)
}

type PeerType string

const (
	PeerBackend  = "b"
	PeerFrontend = "f"
)

func (t PeerType) String() string {
	switch t {
	case PeerBackend:
		return "backend"
	case PeerFrontend:
		return "frontend"
	}
	return string(t)
}

type WsHeader struct {
	Type PeerType `json:"type"`
	Id   string   `json:"id"`
	Name string   `json:"name"` // backend cluster name
}

type Packet struct {
	TargetId     string                     `json:"tid"`
	Sdp          *webrtc.SessionDescription `json:"sdp,omitempty"`
	ICECandidate *webrtc.ICECandidate       `json:"ice,omitempty"`
}

func (s *Server) HandleWs(ctx context.Context, conn *websocket.Conn) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithField("stacks", string(debug.Stack())).Errorf("handle ws paniced:%v", r)
		}
		if conn != nil {
			if err := conn.Close(); err != nil {
				logrus.Errorf("close conn error:%v", err)
			}
		}
		if err != nil {
			logrus.Errorf("handle end with error:%v", err)
		}
	}()

	var header = WsHeader{}
	err = conn.ReadJSON(&header)
	if err != nil {
		return err
	}

	defer func() {
		switch header.Type {
		case PeerBackend:
			s.DelBackend(header.Name, header.Id)
		case PeerFrontend:
			s.DelFrontend(header.Name, header.Id)
		}
	}()

	switch header.Type {
	case PeerBackend:
		s.AddBackend(header.Name, header.Id, conn)
	case PeerFrontend:
		if !s.AddFrontend(header.Name, header.Id, conn) {
			return errors.New("cluster has no backend")
		}
	default:
		return errors.New("unknown header type " + header.Type.String())
	}

	for {
		var packet Packet
		err := conn.ReadJSON(&packet)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		var target *Client
		switch header.Type {
		case PeerBackend:
			target, _ = s.GetFrontend(header.Name, packet.TargetId)
		case PeerFrontend:
			target, _ = s.GetFrontend(header.Name, packet.TargetId)
		}
		if target == nil {
			logrus.Errorf("cannot find target:%s", packet.TargetId)
			continue
		}
		if err := target.Send(packet); err != nil {
			return err
		}
	}
}
