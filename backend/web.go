package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/dgraph-io/badger/v4"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/config"
	"github.com/yixinin/puup/db/file"
	"github.com/yixinin/puup/middles"
	pnet "github.com/yixinin/puup/net"
)

type WebServer struct {
	lis net.Listener
}

func NewWebServer(cfg *config.Config, lis net.Listener) *WebServer {
	return &WebServer{lis: lis}
}

func (s *WebServer) Run(ctx context.Context) error {
	h := &http.Server{}
	h.ConnState = func(c net.Conn, cs http.ConnState) {
		switch cs {
		case http.StateClosed:
			conn, ok := c.(*pnet.Conn)
			if !ok {
				logrus.Error("conn is not *net.Conn")
				return
			}
			conn.Close()
		case http.StateIdle:
			conn, ok := c.(*pnet.Conn)
			if !ok {
				logrus.Error("conn is not *net.Conn")
				return
			}
			conn.Release()
		}
	}
	e := gin.Default()
	e.Use(middles.Cors)
	e.GET("/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "hello there"})
	})

	e.GET("/hello/:id", func(c *gin.Context) {
		var req struct {
			Id int `uri:"id"`
		}
		c.BindUri(&req)
		c.JSON(200, gin.H{
			"msg": "hello there",
			"id":  req.Id,
		})
	})
	e.StaticFS("/share", http.Dir("share"))
	e.GET("/data", SendSerisData)
	e.GET("/opi5", Image)
	initFile(e)
	// e.StaticFS("/share", http.Dir(shareDir))
	e.NoRoute(func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "are you lost?"})
	})
	h.Handler = e

	go h.Serve(s.lis)

	go http.ListenAndServe(":8081", e)

	<-ctx.Done()
	return ctx.Err()
}

type Resp struct {
	Id int `json:"id"`
}

func SendSerisData(c *gin.Context) {
	var s = make([]Resp, 0, 4096)
	for i := 0; i < 1024*4096; i++ {
		s = append(s, Resp{Id: i})
	}

	c.JSON(200, s)
}
func Image(c *gin.Context) {
	f, err := os.Open("share/opi5.png")
	if err != nil {
		c.String(400, "")
		return
	}
	defer f.Close()
	w := c.Writer
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(c.Writer, f)
}

func initFile(e *gin.Engine) {
	g := e.Group("file")

	g.HEAD("/:id", Head)
	g.POST("/upload/pre", PreUpload)
	g.StaticFS("/file", FileSystem{})
}

func PreUpload(c *gin.Context) {
	var req UploadReq
	var ack UploadAck
	var ctx = c.Request.Context()
	if err := c.BindJSON(&req); err != nil {
		c.String(400, err.Error())
		return
	}
	realFile, err := file.GetStorage().GetFile(ctx, req.Etag, req.Size)
	if errors.Is(err, badger.ErrKeyNotFound) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.String(400, err.Error())
		return
	}
	// copy
	uf := file.CopyFile(realFile, req.Path)
	ack.Etag = realFile.Etag
	ack.Path = uf.Path
	c.JSON(200, ack)
	return
}

func Head(c *gin.Context) {
	var ctx = c.Request.Context()
	path := c.Param("id")
	uf, err := file.GetStorage().GetUserFile(ctx, path)
	if errors.Is(err, badger.ErrKeyNotFound) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.String(400, err.Error())
		return
	}

	c.Header("ETAG", uf.Etag)
	c.Header("Content-Length", strconv.FormatUint(uf.Size, 10))
}

type FileSystem struct {
}

func (f FileSystem) Open(name string) (http.File, error) {
	uf, err := file.GetStorage().GetUserFile(context.Background(), name)
	if err != nil {
		return nil, err
	}
	fs, err := os.Open(uf.RealPath)
	return fs, err
}

func Download(c *gin.Context) {
	var ctx = c.Request.Context()
	path := c.Param("id")
	// read range
	var rg = c.Request.Header.Get("Range")
	var start, end int
	var err error
	if rg != "" {
		rgs := strings.Split(rg, "-")
		start, err = strconv.Atoi(rgs[0])
		if err != nil {
			c.String(400, err.Error())
			return
		}
		if len(rgs) > 1 && rgs[1] != "" {
			end, err = strconv.Atoi(rgs[1])
			if err != nil {
				c.String(400, err.Error())
				return
			}
		}
	}
	uf, err := file.GetStorage().GetUserFile(ctx, path)
	if errors.Is(err, badger.ErrKeyNotFound) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.String(400, err.Error())
		return
	}
	c.Header("ETAG", uf.Etag)
	c.Header("Content-Length", strconv.FormatUint(uf.Size-uint64(start), 10))
	var filename = uf.Path
	if isASCII(filename) {
		c.Writer.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	} else {
		c.Writer.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
	}
	fs, err := os.Open(uf.RealPath)
	if err != nil {
		c.String(400, err.Error())
		return
	}
	defer fs.Close()
	if start > 0 {
		_, err := fs.Seek(int64(start), os.SEEK_SET)
		if err != nil {
			c.String(400, err.Error())
			return
		}
	}
	c.Status(200)
	var total int64
	if end > start {
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, uf.Size))
		var buf = make([]byte, 4096)
		for i := start; i < end; i += 4096 {
			n, err := fs.Read(buf)
			if err != nil && err != io.EOF {
				c.String(400, err.Error())
				return
			}
			writen, err := c.Writer.Write(buf[:n])
			if err != nil && err != io.EOF {
				c.String(400, err.Error())
				return
			}
			total += int64(writen)
		}
	} else {
		c.Header("Content-Range", fmt.Sprintf("bytes %d-/%d", start, uf.Size))
	}
	total, err = io.Copy(c.Writer, fs)
	if err != nil {
		logrus.Errorf("write file error")
		return
	}

	return
}
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}
