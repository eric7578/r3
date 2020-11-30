package r3

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"gopkg.in/fsnotify.v1"
)

type PrerendererOption struct {
	Source  string `form:"source" binding:"required"`
	Timeout int    `form:"timeout,default=30"`
	Repeat  int    `form:"repeat,default=1"`
	Cache   int    `form:"cache,default=28800"`
}

type prerenderer struct {
	sync.Mutex
	metaScript string
}

func (r *prerenderer) watchPartial(ctx context.Context, partialDir string) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()

	isOp := func(event fsnotify.Event, ops ...fsnotify.Op) bool {
		for _, op := range ops {
			if event.Op&op == op {
				return true
			}
		}
		return false
	}

	r.reloadPartial(partialDir)

	fatal(w.Add(partialDir))

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.Events:
			if !ok {
				return
			}
			if isOp(event, fsnotify.Write, fsnotify.Remove, fsnotify.Create) {
				r.reloadPartial(partialDir)
			}
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

func (r *prerenderer) reloadPartial(partialDir string) {
	defer r.Unlock()
	r.Lock()

	metaBytes, _ := ioutil.ReadFile(filepath.Join(partialDir, "meta.html"))

	metaTmpl := template.Must(template.New("insertHeadMeta").Parse(jsInsertHeadMeta))

	var metaBuf bytes.Buffer
	metaTmpl.Execute(&metaBuf, struct {
		Meta string
	}{
		Meta: strings.TrimSpace(string(metaBytes)),
	})

	r.metaScript = metaBuf.String()
}

func (r *prerenderer) render(ctx context.Context, opt PrerendererOption) (html string, err error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(opt.Timeout)*time.Second)
	defer cancel()

	for opt.Repeat > 0 {
		html, err = r.fetchPage(ctx, opt.Source)
		if err != nil {
			opt.Repeat--
		} else {
			break
		}
	}

	return
}

func (r *prerenderer) fetchPage(ctx context.Context, u string) (html string, err error) {
	ct, cancel := chromedp.NewContext(ctx)
	defer cancel()

	var res string
	var doc []cdp.NodeID
	err = chromedp.Run(
		ct,
		chromedp.Navigate(u),
		chromedp.NodeIDs("document", &doc, chromedp.ByJSPath),
		chromedp.EvaluateAsDevTools(r.metaScript, &res),
		chromedp.ActionFunc(func(ctx context.Context) error {
			html, err = dom.GetOuterHTML().WithNodeID(doc[0]).Do(ctx)
			return err
		}),
	)

	return
}
