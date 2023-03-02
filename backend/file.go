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
	"github.com/yixinin/puup/storage/file"
)

type FileServer struct {
	lis *pnet.Listener
}

func NewFileServer(cfg *config.Config, lis *pnet.Listener) *FileServer {
	return &FileServer{lis: lis}
}

func (t file.FileType) String() string {
	switch t {
	case file.TypeImage:
		return "image"
	case file.TypeVideo:
		return "video"
	case file.TypeAudio:
		return "audio"
	case file.TypeDoc:
		return "doc"
	}
	return "other"
}

type FileHeader struct {
	Transfer string        `json:"trans"`
	Path     string        `json:"path"`
	FileName string        `json:"file"`
	Size     uint64        `json:"size"`
	FileType file.FileType `json:"type"`
}

type UploadReq struct {
	Path     string        `json:"path"`
	Size     uint64        `json:"size"`
	Etag     string        `json:"etag"`
	FileType file.FileType `json:"type"`
}

type UploadAck struct {
	Code int    `json:"code,omitempty"`
	Path string `json:"path,omitempty"`
	Etag string `json:"etag,omitempty"`
}

type DownloadReq struct {
}
type DownloadAck struct {
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
	var filename = filepath.Join(header.Path, header.FileName)
	switch header.Transfer {
	case "download":
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		_, err = io.Copy(rconn, f)
		if err == io.EOF {
			return nil
		}
		return err
	case "upload":
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

func (f FileServer) upload(r net.Conn, header UploadReq) (UploadAck, error) {
	var ack = UploadAck{}

	return ack, nil
}
