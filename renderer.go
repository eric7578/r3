package r3

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

var (
	reqc chan RenderRequest = make(chan RenderRequest)
)

type RenderRequest struct {
	URL     string
	Retry   int
	resultc chan renderResult
}

type renderResult struct {
	html string
	err  error
}

func startRenderActiivity(ctx context.Context, timeout time.Duration) {
	for {
		req := <-reqc

		ctxAllocator, cancelAllocator := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
		ctxChromedp, cancelChromedp := chromedp.NewContext(ctxAllocator)
		if err := chromedp.Run(ctxChromedp); err != nil {
			log.Fatal(err)
		}

		done := false
		for {
			go renderer(ctxChromedp, &req)

			select {
			case req = <-reqc:
				continue
			case <-time.After(timeout):
				goto WAIT
			case <-ctx.Done():
				done = true
				goto WAIT
			}
		}

	WAIT:
		cancelAllocator()
		cancelChromedp()
		if done {
			return
		}
	}
}

func renderer(ctx context.Context, req *RenderRequest) {
	var doc []cdp.NodeID
	var err error
	var html string

	for req.Retry >= 0 {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			goto DONE

		default:
			if err = chromedp.Run(ctx, chromedp.Tasks{
				chromedp.Navigate(req.URL),
				chromedp.NodeIDs("document", &doc, chromedp.ByJSPath),
				chromedp.ActionFunc(func(ctx context.Context) error {
					if len(doc) == 0 {
						return errors.New("invalid document")
					}
					html, err = dom.GetOuterHTML().WithNodeID(doc[0]).Do(ctx)
					return err
				}),
			}); err != nil {
				req.Retry--
			} else {
				goto DONE
			}
		}
	}

DONE:
	req.resultc <- renderResult{html: html, err: err}
}

func Render(req RenderRequest) (html string, err error) {
	req.resultc = make(chan renderResult)
	go func() {
		reqc <- req
	}()
	res := <-req.resultc
	return res.html, res.err
}
