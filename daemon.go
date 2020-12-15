package r3

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

type DaemonOption struct {
	ConfigDir string
}

type Daemon struct {
	pre *prerenderer
}

func NewDaemon(opt DaemonOption) *Daemon {
	r := &prerenderer{
		configDir:   opt.ConfigDir,
		metaScripts: make(map[string]string),
	}
	go r.watchConfigFiles(context.TODO())
	return &Daemon{
		pre: r,
	}
}

func (d *Daemon) Run(port string) {
	r := gin.Default()
	r.GET("/", d.renderHandler)
	r.Run(port)
}

type RenderOption struct {
	Source            string `form:"src" binding:"required"`
	ExternalResources bool   `form:"extres,default=true"`
	EmbeddedCSS       bool   `form:"embcss,default=false"`
	Timeout           int    `form:"timeout,default=30"`
	Repeat            int    `form:"repeat,default=1"`
}

func (d *Daemon) renderHandler(c *gin.Context) {
	var opt RenderOption
	if err := c.BindQuery(&opt); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	// using prerenderers
	html, err := d.pre.render(c.Request.Context(), opt)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			c.String(http.StatusRequestTimeout, "request timeout")
		default:
			panic(err)
		}
		return
	}
	bytes := []byte(html)
	c.Data(http.StatusOK, gin.MIMEHTML, bytes)
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}
