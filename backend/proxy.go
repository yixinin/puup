package backend

import (
	"fmt"
	"io"
	"net"

	"github.com/yixinin/puup/pnet"
	"github.com/yixinin/puup/stderr"
)

type Proxy struct {
	ports map[uint16]struct{}
}

func NewProxy(port ...uint16) (*Proxy, error) {
	var ports = make(map[uint16]struct{})
	for _, v := range port {
		ports[v] = struct{}{}
	}
	return &Proxy{
		ports: ports,
	}, nil
}

func (p *Proxy) Serve(rconn net.Conn) error {
	defer func() {
		conn, ok := rconn.(*pnet.Conn)
		if ok {
			conn.Release()
			conn.Close()
		}
	}()
	port := rconn.RemoteAddr().(*pnet.LabelAddr).ProxyPort()
	if _, ok := p.ports[port]; !ok {
		return stderr.New("invalid port proxy")
	}
	lconn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	err = func(src, dst net.Conn) error {
		var errChan = make(chan error, 1)
		go func() {
			_, err := io.Copy(src, dst)
			if err == nil || err == io.EOF {
				errChan <- nil
				return
			}
			errChan <- err
		}()
		go func() {
			_, err := io.Copy(dst, src)
			if err == nil || err == io.EOF {
				errChan <- nil
				return
			}
			errChan <- err
		}()
		for i := 0; i < 2; i++ {
			if err := <-errChan; err != nil {
				return err
			}
		}
		return nil
	}(lconn, rconn)
	return err
}
