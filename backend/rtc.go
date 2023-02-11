package backend

import (
	"fmt"

	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/x264"
	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/ice"
	"github.com/yixinin/puup/pnet"
)

type RtcServer struct {
	serverAddr, backendName string
	Screen                  *webrtc.PeerConnection
	Camera                  *webrtc.PeerConnection
}

func NewRtcServer(serverAddr, backendName string) *RtcServer {
	return &RtcServer{
		serverAddr:  serverAddr,
		backendName: backendName,
	}
}
func (s *RtcServer) loop() {

}

func (s *RtcServer) RunCamera() error {
	var backendName = fmt.Sprintf("%s.camera", s.backendName)
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return err
	}
	sigCli := pnet.NewAnswerClient(s.serverAddr, backendName)
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

	peer := pnet.NewAnswerPeer(pc, sigCli, nil)
	if err := peer.Connect(); err != nil {
		return err
	}
	return nil
}

func (s *RtcServer) RunScreen() error {
	var backendName = fmt.Sprintf("%s.screen", s.backendName)
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return err
	}
	sigCli := pnet.NewAnswerClient(s.serverAddr, backendName)

	peer := pnet.NewAnswerPeer(pc, sigCli, nil)
	if err := peer.Connect(); err != nil {
		return err
	}
	return nil
}
