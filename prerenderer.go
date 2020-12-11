package r3

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/radovskyb/watcher"
)

type PrerendererOption struct {
	Source  string `form:"source" binding:"required"`
	Timeout int    `form:"timeout,default=30"`
	Repeat  int    `form:"repeat,default=1"`
	Cache   int    `form:"cache,default=28800"`
}

type prerenderer struct {
	sync.Mutex
	configDir        string
	metaScripts      map[string]string
	exteralResources bool
}

func (r *prerenderer) watchConfigFiles(ctx context.Context) {
	w := watcher.New()
	w.FilterOps(watcher.Create, watcher.Write)
	defer w.Close()

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ctx.Done():
				goto END
			case event := <-w.Event:
				if !event.IsDir() && filepath.Ext(event.Name()) == ".html" {
					r.reloadMetaFile(event.Path)
				}
			}
		}

	END:
		done <- struct{}{}
	}()

	fatal(w.AddRecursive(r.configDir))
	fatal(filepath.Walk(r.configDir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".html" {
			r.reloadMetaFile(path)
		}
		return nil
	}))
	fatal(w.Start(time.Millisecond * 1000))
	<-done
}

func (r *prerenderer) reloadMetaFile(metaPath string) {
	defer r.Unlock()
	r.Lock()

	relPath, _ := filepath.Rel(r.configDir, metaPath)
	urlPath := filepath.Dir(relPath)
	if urlPath == "." {
		urlPath = "/"
	} else {
		urlPath = "/" + urlPath
	}

	metaBytes, _ := ioutil.ReadFile(metaPath)
	metaTmpl := template.Must(template.New("insertHeadMeta").Parse(jsInsertHeadMeta))

	var metaBuf bytes.Buffer
	metaTmpl.Execute(&metaBuf, struct {
		Meta string
	}{
		Meta: strings.TrimSpace(string(metaBytes)),
	})

	r.metaScripts[urlPath] = metaBuf.String()
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

	return r.postRender(html)
}

func (r *prerenderer) fetchPage(ctx context.Context, rawurl string) (html string, err error) {
	ct, cancel := chromedp.NewContext(ctx)
	defer cancel()

	var res string
	var doc []cdp.NodeID
	actions := []chromedp.Action{
		chromedp.Navigate(rawurl),
		chromedp.NodeIDs("document", &doc, chromedp.ByJSPath),
		chromedp.EvaluateAsDevTools(r.metaScripts["/"], &res),
	}

	u, _ := url.Parse(rawurl)
	metaScript := r.metaScripts[u.Path]
	if metaScript != "" {
		var resAdditional string
		actions = append(actions, chromedp.EvaluateAsDevTools(metaScript, &resAdditional))
	}

	actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
		html, err = dom.GetOuterHTML().WithNodeID(doc[0]).Do(ctx)
		return err
	}))

	err = chromedp.Run(ct, actions...)
	return
}

func (r *prerenderer) postRender(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}
	if !r.exteralResources {
		removeExternalResources(doc)
	}
	return doc.Html()
}

func removeExternalResources(doc *goquery.Document) {
	doc.Find("script,link[rel='stylesheet']").Remove()
}
