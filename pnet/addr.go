package pnet

import (
	"fmt"
	"net"
)

type LabelAddr struct {
	Name  string
	Label string
}

func NewLabelAddr(name, label string) net.Addr {
	return &LabelAddr{
		Name:  name,
		Label: label,
	}
}

func (a *LabelAddr) Network() string {
	return "webrtc"
}

func (a *LabelAddr) String() string {
	return fmt.Sprintf("%s:%s", a.Name, a.Label)
}
