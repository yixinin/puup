package backend

import (
	"fmt"

	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/x264"
	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/ice"
	"github.com/yixinin/puup/net"
)

type RtcServer struct {
	sigAddr, sigAddr string
	Screen           *webrtc.PeerConnection
	Camera           *webrtc.PeerConnection
}

func NewRtcServer(sigAddr, sigAddr string) *RtcServer {
	return &RtcServer{
		sigAddr: sigAddr,
		sigAddr: sigAddr,
	}
}
func (s *RtcServer) loop() {

}

func (s *RtcServer) RunCamera() error {
	var sigAddr = fmt.Sprintf("%s.camera", s.sigAddr)
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return err
	}
	sigCli := net.NewAnswerClient(s.sigAddr, sigAddr)
	stream, err := mediadevices.GetDisplayMedia(mediadevices.MediaStreamConstraints{
		Video: func(mtc *mediadevices.MediaTrackConstraints) {

		},
		Audio: func(mtc *mediadevices.MediaTrackConstraints) {

		},
		Codec: mediadevices.NewCodecSelector(mediadevices.WithAudioEncoders(x264.NewParams())),
	})
	for _, v := range stream.GetTracks() {
		pc.AddTrack(v)
	}

	peer := net.NewAnswerPeer(pc, sigCli, nil)
	if err := peer.Connect(); err != nil {
		return err
	}
	return nil
}

func (s *RtcServer) RunScreen() error {
	var sigAddr = fmt.Sprintf("%s.screen", s.sigAddr)
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return err
	}
	sigCli := net.NewAnswerClient(s.sigAddr, sigAddr)

	peer := net.NewAnswerPeer(pc, sigCli, nil)
	if err := peer.Connect(); err != nil {
		return err
	}
	return nil
}
