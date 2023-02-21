package backend

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/yixinin/puup/config"
	pnet "github.com/yixinin/puup/net"
)

type FileServer struct {
	lis *pnet.Listener
}

func NewFileServer(cfg *config.Config, lis *pnet.Listener) *FileServer {
	return &FileServer{lis: lis}
}

type FileHeader struct {
	Type     string `json:"type"`
	Path     string `json:"path"`
	Filename string `json:"filename"`
}

func (s *FileServer) Run(ctx context.Context) error {
	for {
		conn, err := s.lis.AcceptFile()
		if err != nil {
			return err
		}
		err = s.ServeConn(ctx, conn)
		if err != nil {
			return err
		}
	}
}

func (s *FileServer) ServeConn(ctx context.Context, rconn net.Conn) error {
	defer func() {
		rconn.(*pnet.Conn).Release()
	}()
	var header FileHeader
	data, _, err := bufio.NewReader(rconn).ReadLine()
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &header)
	if err != nil {
		return err
	}
	var filename = filepath.Join(header.Path, header.Filename)
	switch header.Type {
	case "pull":
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		_, err = io.Copy(rconn, f)
		if err == io.EOF {
			return nil
		}
		return err
	case "push":
		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, rconn)
		if err == io.EOF {
			return nil
		}
		return err
	}
	return nil
}
