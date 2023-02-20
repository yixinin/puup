package server

import (
	"github.com/gin-gonic/gin"
	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/proto"
)

func (s *Server) Fetch(c *gin.Context) {
	var req proto.FetchReq
	c.BindQuery(&req)
	var ack = new(proto.FetchAck)

	defer c.JSON(200, ack)

	b := s.GetBackend(req.Name)
	var sess *Session
	if req.Id != "" {
		sess = b.GetSession(req.Id)
	}

	sess = b.RandSession()
	if sess == nil {
		return
	}
	if sess.IsClose() {
		return
	}
	fetchSession(req.Type, sess, ack)
}

func fetchSession(typ webrtc.SDPType, sess *Session, ack *proto.FetchAck) {
	switch typ {
	case webrtc.SDPTypeOffer:
		for {
			select {
			case sdp := <-sess.answer:
				ack.Sdp = &sdp
			case ice := <-sess.answerIce:
				ack.Candidates = append(ack.Candidates, ice)
			default:
				return
			}
		}

	case webrtc.SDPTypeAnswer:
		for {
			select {
			case sdp := <-sess.offer:
				ack.Sdp = &sdp
			case ice := <-sess.offerIce:
				ack.Candidates = append(ack.Candidates, ice)
			default:
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
