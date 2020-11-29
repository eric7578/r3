package r3

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

type Daemon struct {
}

func NewDaemon() *Daemon {
	return &Daemon{}
}

func (d *Daemon) Run(port string) {
	r := gin.Default()
	r.GET("/prerender", prerenderHandler)
	r.Run(port)
}

func prerenderHandler(c *gin.Context) {
	u := c.Query("url")
	if !isURL(u) {
		err := errors.New("invalid prerenderer url")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"url": u,
	})
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
