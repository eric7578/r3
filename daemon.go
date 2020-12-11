package r3

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

type Daemon struct {
	renderer *prerenderer
}

func NewDaemon(configDir string) *Daemon {
	r := &prerenderer{
		configDir:   configDir,
		metaScripts: make(map[string]string),
	}
	go r.watchConfigFiles(context.TODO())
	return &Daemon{
		renderer: r,
	}
}

func (d *Daemon) Run(port string) {
	r := gin.Default()
	r.GET("/prerender", d.prerenderHandler)
	r.Run(port)
}

func (d *Daemon) prerenderHandler(c *gin.Context) {
	var opt PrerendererOption
	if err := c.BindQuery(&opt); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	} else if !isURL(opt.Source) {
		c.String(http.StatusBadRequest, "invalid prerenderer source url")
		return
	}

	// using prerenderer
	html, err := d.renderer.render(c.Request.Context(), opt)
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
