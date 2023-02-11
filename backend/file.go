package backend

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/yixinin/puup/pnet"
)

type FileServer struct {
}

func NewFileServer() *FileServer {
	return &FileServer{}
}

type FileHeader struct {
	Type     string `json:"type"`
	Path     string `json:"path"`
	Filename string `json:"filename"`
}

func (s *FileServer) Serve(conn net.Conn) error {
	defer func() {
		conn.(*pnet.Conn).Release()
	}()
	var header FileHeader
	data, _, err := bufio.NewReader(conn).ReadLine()
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
		_, err = io.Copy(conn, f)
		if err == io.EOF {
			return nil
		}
		return err
	case "push":
		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, conn)
		if err == io.EOF {
			return nil
		}
		return err
	}
	return nil
}
