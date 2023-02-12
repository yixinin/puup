package proto

import "github.com/pion/webrtc/v3"

type PostSdpReq struct {
	Name string                    `json:"name"`
	Key  string                    `json:"key"`
	Sdp  webrtc.SessionDescription `json:"sdp"`
}
