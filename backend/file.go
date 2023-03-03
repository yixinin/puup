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
	"github.com/sirupsen/logrus"
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

type UploadReq struct {
	Path     string        `json:"path"`
	Size     uint64        `json:"size"`
	Etag     string        `json:"etag"`
	FileType file.FileType `json:"type"`
	StartAt  uint64        `json:"start"`
}

type UploadAck struct {
	Code int    `json:"code,omitempty"`
	Path string `json:"path,omitempty"`
	Etag string `json:"etag,omitempty"`
}

type DownloadReq struct {
}
type DownloadAck struct {
	Etag string
	Size string
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
	var req UploadReq
	data, _, err := bufio.NewReader(rconn).ReadLine()
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &req)
	if err != nil {
		return err
	}

	ack, err := s.upload(ctx, rconn, req)
	if err != nil {
		logrus.Error("upload failed:%v", err)
	}
	data, _ = json.Marshal(ack)
	_, err = rconn.Write(data)
	return err
}

func (f *FileServer) upload(ctx context.Context, r net.Conn, req UploadReq) (UploadAck, error) {
	var ack = UploadAck{}
	// check
	rf, err := file.GetStorage().GetFile(ctx, req.Etag, req.Size)
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return ack, err
	}

	// exists, copy file
	if err == nil {
		uf := file.CopyFile(rf, req.Path)
		err = file.GetStorage().InsertUserFile(ctx, uf)
		if err != nil {
			return ack, err
		}
		ack.Etag = req.Etag
		ack.Path = req.Path
		return ack, nil
	}
	// new file
	var filename = file.GetFileName(req.Etag, req.Size, filepath.Ext(req.Path))
	var realFile = file.File{
		Etag: req.Etag,
		Type: req.FileType,
		Size: req.Size,
		Path: filename,
	}
	err = file.GetStorage().InsertFile(ctx, realFile)
	if err != nil {
		return ack, err
	}

	var fs *os.File

	if req.StartAt == 0 {
		fs, err = os.Create(filename)
	} else {
		fs, err = os.Open(filename)
		if err != nil {
			return ack, err
		}
		_, err = fs.Seek(int64(req.StartAt), os.SEEK_SET)
	}
	if err != nil {
		return ack, err
	}
	defer fs.Close()
	_, err = io.Copy(fs, r)
	if err != nil {
		return ack, err
	}

	err = file.GetStorage().InsertUserFile(ctx, file.CopyFile(realFile, req.Path))
	if err != nil {
		return ack, err
	}
	ack.Etag = req.Etag
	ack.Path = req.Path
	return ack, nil
}
