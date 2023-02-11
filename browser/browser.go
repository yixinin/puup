package browser

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yixinin/puup/middles"
)

func RunBrowser() {
	e := gin.Default()
	e.Use(middles.Cors)
	e.StaticFS("/", http.Dir("dist"))
	http.ListenAndServe(":8080", e)
}
