package backend

import (
	"context"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/pnet"
)

type WebServer struct {
}

func RunServer(ctx context.Context, name, puup, shareDir string) error {
	ln := pnet.NewListener(name, puup)
	defer ln.Close()

	h := &http.Server{}
	h.ConnState = func(c net.Conn, cs http.ConnState) {
		switch cs {
		case http.StateClosed:
			conn, ok := c.(*pnet.Conn)
			if !ok {
				logrus.Error("conn is not *pnet.Conn")
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
	e.StaticFS("/share", http.Dir(shareDir))
	e.NoRoute(func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "are you lost?"})
	})
	h.Handler = e

	go h.Serve(ln)

	<-ctx.Done()
	return ctx.Err()
}
