package conn

import (
	"net"
)

func (p *Peer) Read(data []byte) (int, error) {
	select {
	case <-p.close:
		return 0, net.ErrClosed
	case buf := <-p.recvData:
		if len(data) < len(buf) {
			panic("read out of memeroy")
		}
		n := copy(data, buf)
		return int(n), nil
	}
}

func (p *Peer) Write(data []byte) (int, error) {
	<-p.open
	err := p.data.Send(data)
	return len(data), err
}
