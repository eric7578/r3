package r3

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

type Daemon struct {
	cache    *cache.Cache
	renderer *prerenderer
}

func NewDaemon() *Daemon {
	return &Daemon{
		cache:    cache.New(8*time.Hour, 12*time.Hour),
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
		return
	} else if !isURL(opt.Source) {
		c.String(http.StatusBadRequest, "invalid prerenderer source url")
		return
	}

	// using cache
	if _, found := d.cache.Get(opt.Source); found {
		c.String(http.StatusNotModified, "")
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
	d.cache.Set(opt.Source, true, time.Duration(opt.Cache)*time.Second)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}
