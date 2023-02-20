package frontend

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/config"
	pnet "github.com/yixinin/puup/net"
)

type CopyFile struct {
	localName  string
	remoteName string
	mode       string
}

type FileClient struct {
	serverName string
	sigAddr    string

	ch chan CopyFile
}

func NewFileClient(cfg *config.Config) *FileClient {
	return &FileClient{
		serverName: fmt.Sprintf("%s.file", cfg.ServerName),
		sigAddr:    cfg.SigAddr,
		ch:         make(chan CopyFile, 1),
	}
}

func (c *FileClient) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case file, ok := <-c.ch:
			if !ok {
				return net.ErrClosed
			}
			err := c.handle(file)
			if err != nil {
				logrus.Errorf("scp file error:%v", err)
			}
		}
	}
}

func (c *FileClient) handle(file CopyFile) error {
	conn, err := pnet.Dial(c.sigAddr, c.serverName)
	if err != nil {
		return err
	}

	switch file.mode {
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
		f, err := os.Open(file.localName)
		if err != nil {
			return err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return c.CopyFs(conn, f, file.remoteName)
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

func (c *FileClient) CopyFile(conn net.Conn, src, dst string) error {
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

func (c *FileClient) CopyFs(conn net.Conn, src *os.File, dst string) error {
	conn.Write([]byte(dst))
	conn.Write([]byte{'\n'})
	_, err := io.Copy(conn, src)
	return err
}
