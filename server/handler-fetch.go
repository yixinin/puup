package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/proto"
)

func (s *Server) Fetch(c *gin.Context) {
	var req proto.FetchReq
	c.BindQuery(&req)
	var ack = new(proto.FetchAck)
	defer func() {
		if len(ack.Candidates) == 0 && ack.Sdp == nil {
			c.String(203, "")
		} else {
			c.JSON(200, ack)
		}
	}()
	b := s.GetBackend(req.Name)
	var sess *Session
	if req.Id != "" {
		sess = b.GetSession(req.Id)
	} else {
		sess = b.RandSession()
	}

	if sess == nil {
		return
	}
	req.Id = sess.Id
	ack.Id = sess.Id
	if sess.IsClose() {
		logrus.Debugf("session %s %s %s is close", req.Name, req.Id, req.Type)
		return
	}
	fetchSession(req.Type, sess, ack)
}

func fetchSession(typ webrtc.SDPType, sess *Session, ack *proto.FetchAck) {
	var t = time.NewTimer(time.Second)
	defer t.Stop()
	switch typ {
	case webrtc.SDPTypeOffer:
		for {
			select {
			case sdp := <-sess.answer:
				logrus.Debugf("fetch %s sdp", sdp.Type)
				ack.Sdp = &sdp
			case ice := <-sess.answerIce:
				ack.Candidates = append(ack.Candidates, ice)
			case <-t.C:
				return
			}
		}

	case webrtc.SDPTypeAnswer:
		for {
			select {
			case sdp := <-sess.offer:
				logrus.Debugf("fetch %s sdp", sdp.Type)
				ack.Sdp = &sdp
			case ice := <-sess.offerIce:
				ack.Candidates = append(ack.Candidates, ice)
			case <-t.C:
				return
			}
		}
	}
}

func (s *Server) Offline(c *gin.Context) {
	var req proto.OfflineReq
	c.BindQuery(&req)
	b := s.GetBackend(req.Name)
	b.DelSession(req.Id)
}
