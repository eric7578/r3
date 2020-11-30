package r3

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

type Daemon struct {
	cache    *cache.Cache
	renderer *prerenderer
}

func NewDaemon(configDir string) *Daemon {
	r := &prerenderer{}
	go r.watchPartial(context.TODO(), filepath.Join(configDir, "partial"))
	return &Daemon{
		cache:    cache.New(8*time.Hour, 12*time.Hour),
		renderer: r,
	}
}

func (d *Daemon) Run(port string) {
	r := gin.Default()
	r.GET("/prerender", d.prerenderHandler)
	r.DELETE("/prerender", d.clearCache)
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
	if htmlCache, found := d.cache.Get(opt.Source); found {
		c.Data(http.StatusOK, gin.MIMEHTML, htmlCache.([]byte))
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
	d.cache.Set(opt.Source, bytes, time.Duration(opt.Cache)*time.Second)
	c.Data(http.StatusOK, gin.MIMEHTML, bytes)
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func (d *Daemon) clearCache(c *gin.Context) {
	type Body struct {
		Source string `json:"source"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	if body.Source == "" {
		d.cache.Flush()
	} else {
		d.cache.Delete(body.Source)
	}
	c.Status(http.StatusOK)
}

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}
