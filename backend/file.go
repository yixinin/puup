package backend

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"
	"github.com/yixinin/puup/config"
	"github.com/yixinin/puup/db/file"
	pnet "github.com/yixinin/puup/net"
)

type FileServer struct {
	lis *pnet.Listener
}

func NewFileServer(cfg *config.Config, lis *pnet.Listener) *FileServer {
	return &FileServer{lis: lis}
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

func (f *FileServer) upload(ctx context.Context, r net.Conn, req UploadReq) (UploadAck, error) {
	var ack = UploadAck{}
	// check
	realFile, err := file.GetStorage().GetFile(ctx, req.Etag, req.Size)
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return ack, err
	}

	return ack, nil
}
