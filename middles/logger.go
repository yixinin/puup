package middles

import (
	"bytes"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const MAX_PRINT_BODY_LEN = 512

type bodyLogWriter struct {
	gin.ResponseWriter
	bodyBuf *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	//memory copy here!
	w.bodyBuf.Write(b)
	return w.ResponseWriter.Write(b)
}
func (r bodyLogWriter) Read(b []byte) (int, error) {
	return r.bodyBuf.Read(b)
}
func (rw bodyLogWriter) Close() error {
	return nil
}

func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		var blw bodyLogWriter
		//if we need to log res body
		var isFetch = strings.Contains(c.Request.RequestURI, "fetch")
		if !isFetch {
			var buf []byte
			if c.Request.Body != nil {
				buf, _ = io.ReadAll(c.Request.Body)
			}
			logrus.WithField("url", c.Request.URL.Path).
				WithField("form", c.Request.URL.Query()).
				WithField("addr", c.Request.RemoteAddr).
				Info("incoming request", string(buf))

			blw = bodyLogWriter{bodyBuf: bytes.NewBuffer(buf), ResponseWriter: c.Writer}
			c.Request.Body = blw
			c.Writer = blw
		}

		c.Next()

		if !isFetch || c.Writer.Status() == 200 {
			strBody := strings.Trim(blw.bodyBuf.String(), "\n")
			if len(strBody) > MAX_PRINT_BODY_LEN {
				strBody = strBody[:(MAX_PRINT_BODY_LEN - 1)]
			}
			logrus.WithField("url", c.Request.URL.Path).Infof("outgoing response: %s", strBody)
		}
	}
}
