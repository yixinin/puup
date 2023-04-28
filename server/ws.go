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
	"github.com/yixinin/puup/proto"
)

func (s *Server) WsSignalling(c *gin.Context) {
	conn, err := s.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.String(http.StatusBadRequest, "upgrade failed, error:%v", err)
	}
	go s.HandleWs(c.Request.Context(), conn)
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

	var header = proto.WsHeader{}
	err = conn.ReadJSON(&header)
	if err != nil {
		return err
	}

	defer func() {
		switch header.Type {
		case webrtc.SDPTypeAnswer:
			s.DelBackend(header.Name, header.Id)
		case webrtc.SDPTypeOffer:
			s.DelFrontend(header.Name, header.Id)
		}
	}()

	switch header.Type {
	case webrtc.SDPTypeAnswer:
		s.AddBackend(header.Name, header.Id, conn)
	case webrtc.SDPTypeOffer:
		if !s.AddFrontend(header.Name, header.Id, conn) {
			return errors.New("cluster has no backend")
		}
	default:
		return errors.New("unknown header type " + header.Type.String())
	}

	for {
		var packet proto.Packet
		err := conn.ReadJSON(&packet)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		var client *Client
		switch header.Type {
		case webrtc.SDPTypeAnswer:
			client, _ = s.GetFrontend(header.Name, packet.From.ClientId)
		case webrtc.SDPTypeOffer:
			client, _ = s.GetBackend(header.Name, packet.From.ClientId)
		}
		if client == nil {
			logrus.Errorf("cannot find target:%s", packet.From.ClientId)
			continue
		}

		packet.To.ClientId = client.Peers[packet.From.PeerId]

		if err := client.Send(packet); err != nil {
			return err
		}
	}
}
