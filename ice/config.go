package ice

import "github.com/pion/webrtc/v3"

var Config = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:114.115.218.1:3478"},
		},
	},
}
