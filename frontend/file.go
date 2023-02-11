package frontend

import (
	"bufio"
	"io"
	"net"
	"os"

	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/ice"
	"github.com/yixinin/puup/pnet"
)

type ScpClient struct {
	localFilename, remoteFilename, name string
	mode                                string
}

func NewScpClient(localFilename, remoteFilename string, name string) *ScpClient {
	return &ScpClient{
		localFilename:  localFilename,
		remoteFilename: remoteFilename,
		name:           name,
	}
}

func (c *ScpClient) Run() error {
	var pc, err = webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return err
	}
	sigCli := pnet.NewOfferClient(defaultpuup, c.name)
	p, err := pnet.NewOfferPeer(pc, sigCli)
	if err != nil {
		return err
	}
	if err = p.Connect(); err != nil {
		return err
	}
	conn, err := p.GetWebConn("")
	if err != nil {
		return err
	}

	switch c.mode {
	case "pull":
		for {
			rd := bufio.NewReader(conn)
			line, _, err := rd.ReadLine()
			if err != nil {
				return err
			}
			filename := string(line)
			for {
				err := func() error {
					f, err := os.Create(filename)
					if err != nil {
						return err
					}
					if _, err := io.Copy(f, conn); err != nil && err != io.EOF {
						return err
					}
					return err
				}()
				if err != nil {
					return err
				}
			}
		}
	case "push":
		f, err := os.Open(c.localFilename)
		if err != nil {
			return err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return c.CopyFs(conn, f, c.remoteFilename)
		}
		fs, err := f.ReadDir(-1)
		if err != nil {
			return err
		}
		for _, e := range fs {
			if err := c.CopyFile(conn, e.Name(), e.Name()); err != nil {
				return err
			}
		}
	}

	return nil
}
func (c *ScpClient) CopyFile(conn net.Conn, src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	conn.Write([]byte(dst))
	conn.Write([]byte{'\n'})
	_, err = io.Copy(conn, f)
	return err
}

func (c *ScpClient) CopyFs(conn net.Conn, src *os.File, dst string) error {
	conn.Write([]byte(dst))
	conn.Write([]byte{'\n'})
	_, err := io.Copy(conn, src)
	return err
}
