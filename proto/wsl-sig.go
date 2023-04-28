package proto

import "github.com/pion/webrtc/v3"

type WsHeader struct {
	Type webrtc.SDPType `json:"type"`
	Id   string         `json:"id"`
	Name string         `json:"name"` // backend cluster name
}

type Client struct {
	ClientId string `json:"cid"`
	PeerId   string `json:"pid"`
}

type Packet struct {
	From         Client                     `json:"from"`
	To           Client                     `json:"to"`
	Sdp          *webrtc.SessionDescription `json:"sdp,omitempty"`
	ICECandidate *webrtc.ICECandidate       `json:"ice,omitempty"`
}
