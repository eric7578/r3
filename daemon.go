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

func NewDaemon() *Daemon {
	return &Daemon{
		renderer: &prerenderer{},
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
	} else if !isURL(opt.Source) {
		c.String(http.StatusBadRequest, "invalid prerenderer source url")
	} else {
		if html, err := d.renderer.render(c.Request.Context(), opt); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				c.String(http.StatusRequestTimeout, "request timeout")
			} else {
				panic(err)
			}
		} else {
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
		}
	}
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
