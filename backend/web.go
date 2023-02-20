package backend

import (
	"context"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/config"
	pnet "github.com/yixinin/puup/net"
)

type WebServer struct {
	lis net.Listener
}

func NewWebServer(cfg *config.Config) *WebServer {
	lis := pnet.NewListener(cfg.SigAddr, cfg.ServerName)
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
			conn.Release()
		}
	}
	e := gin.Default()

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
	// e.StaticFS("/share", http.Dir(shareDir))
	e.NoRoute(func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "are you lost?"})
	})
	h.Handler = e

	go h.Serve(s.lis)

	<-ctx.Done()
	return ctx.Err()
}
