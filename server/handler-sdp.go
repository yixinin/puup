package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/proto"
)

func (s *Server) PostSdp(c *gin.Context) {
	var req proto.PostSdpReq
	c.MustBindWith(&req, binding.JSON)

	b := s.GetBackend(req.Name)
	sess := b.MustGetSession(req.Id)
	if sess.IsClose() {
		c.String(200, "session closed")
		return
	}

	defer c.String(200, "")
	switch req.Sdp.Type {
	case webrtc.SDPTypeOffer:
		sess.offer <- req.Sdp
		logrus.Debugf("receive %s %s offer", req.Name, req.Id)
	case webrtc.SDPTypeAnswer:
		sess.answer <- req.Sdp
		logrus.Debugf("receive %s %s answer", req.Name, req.Id)
	}
}
