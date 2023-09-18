package rtc

import "github.com/pion/webrtc/v3"

type ConnPool struct {
	ClusterName string
	Conns       map[string]webrtc.PeerConnection
}

func NewConnPool(clusterName string) {

}
