package r3

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Daemon struct {
	RendererAwake time.Duration
}

func (d *Daemon) Run(ctx context.Context, port string) {
	go startRenderActiivity(ctx, d.RendererAwake)

	r := gin.Default()
	r.POST("/", d.renderHandler)
	r.Run(port)
}

func (d *Daemon) renderHandler(c *gin.Context) {
	type RenderBody struct {
		URL string `json:"url" binding:"required"`
	}

	var body RenderBody
	if err := c.BindJSON(&body); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	// using prerenderers
	html, err := Render(RenderRequest{URL: body.URL})
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
