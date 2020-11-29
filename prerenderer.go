package r3

import (
	"context"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

type PrerendererOption struct {
	Source  string `form:"source" binding:"required"`
	Timeout int    `form:"timeout"`
	Ignores string `form:"ignores"`
	Repeat  int    `form:"repeat"`
}

type prerenderer struct {
	PrerendererOption
}

func (r *prerenderer) render(ctx context.Context, opt PrerendererOption) (html string, err error) {
	if opt.Timeout == 0 {
		opt.Timeout = 30
	}
	timeout := time.Duration(opt.Timeout) * time.Second

	repeat := 1
	if opt.Repeat > repeat {
		repeat = opt.Repeat
	}

	// ignoreElements := strings.Split(opt.Ignores, ",")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for repeat > 0 {
		html, err = r.fetchPage(ctx, opt.Source)
		if err != nil {
			repeat -= 1
		} else {
			break
		}
	}

	return
}

func (r *prerenderer) fetchPage(ctx context.Context, u string) (html string, err error) {
	ct, cancel := chromedp.NewContext(ctx)
	defer cancel()

	var doc []cdp.NodeID
	err = chromedp.Run(
		ct,
		chromedp.Navigate(u),
		chromedp.NodeIDs("document", &doc, chromedp.ByJSPath),
		chromedp.ActionFunc(func(ctx context.Context) error {
			html, err = dom.GetOuterHTML().WithNodeID(doc[0]).Do(ctx)
			return err
		}),
	)

	return
}
