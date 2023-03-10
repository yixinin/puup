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
	"github.com/yixinin/puup/preview"
	"github.com/yixinin/puup/stderr"
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
	realFile, err := file.GetStorage().GetFile(ctx, req.Etag, req.Size)
	if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
		return ack, err
	}
	var fs *os.File
	var previewPath, filename = file.GetFileName(req.Etag, req.Size, filepath.Ext(req.Path))
	if req.StartAt == 0 {
		// exists, copy file
		if err == nil {
			uf := file.CopyFile(realFile, req.Path)
			err = file.GetStorage().InsertUserFile(ctx, uf)
			if err != nil {
				return ack, err
			}
			ack.Etag = req.Etag
			ack.Path = req.Path
			return ack, nil
		}

		// new file
		realFile = file.File{
			Etag: req.Etag,
			Type: req.FileType,
			Size: req.Size,
			Path: filename,
		}
		if req.FileType == file.TypeImage || req.FileType == file.TypeVideo {
			realFile.PreviewPath = previewPath
		}
		err = file.GetStorage().InsertFile(ctx, realFile)
		if err != nil {
			return ack, err
		}
		fs, err = os.Create(filename)
	} else {
		if err != nil {
			return ack, stderr.Wrap(err)
		}
		fs, err = os.Open(filename)
		if err != nil {
			return ack, stderr.Wrap(err)
		}
		_, err = fs.Seek(int64(req.StartAt), io.SeekStart)
	}
	if err != nil {
		return ack, stderr.Wrap(err)
	}

	defer fs.Close()
	_, err = io.Copy(fs, r)
	if err != nil {
		return ack, err
	}
	if err := fs.Close(); err != nil {
		return ack, stderr.Wrap(err)
	}

	switch req.FileType {
	case file.TypeImage:
		err = preview.SaveImagePreview(filename, previewPath)
	case file.TypeVideo:
		err = preview.SaveVideoPreview(filename, previewPath, 60)
	}
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
