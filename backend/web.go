package backend

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/config"
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
	e.POST("/file/upload", func(c *gin.Context) {
		fielname := c.Request.Header.Get("filename")
		fielname = filepath.Join("share", fielname)
		del := c.Request.Header.Get("del")
		if _, err := os.Stat(fielname); err != os.ErrExist {
			if del != "del" {
				c.String(400, "file already exsit")
				return
			}
			os.Remove(fielname)
		}

		f, err := os.Create(fielname)
		if err != nil {
			c.String(400, err.Error())
			return
		}
		defer c.Request.Body.Close()
		_, err = io.Copy(f, c.Request.Body)
		if err != nil {
			c.String(400, err.Error())
			return
		}
		c.String(200, "")
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
