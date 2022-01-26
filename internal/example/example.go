package example

import (
	"github.com/ivangodev/fefa/pkg/fefa"
	"sync"
)

var Results []interface{}
var ResultsMu sync.Mutex

type FetchCallbacks struct {
	PagesFetch func(page int) (validPage bool)
	UrlsFetch  func(page int) (URLs []string)
	UrlFetch   func(URL string) (data interface{})
}

type PagesFeFa struct {
	currPage int
	Cb       FetchCallbacks
}

func (f *PagesFeFa) Prepare() {
	f.currPage = 0
	Results = make([]interface{}, 0)
}

func (f *PagesFeFa) Next() fefa.FetcherFast {
	f.currPage++
	if f.Cb.PagesFetch(f.currPage) {
		return &URLsFeFa{page: f.currPage, cb: f.Cb}
	}
	return nil
}

func (f *PagesFeFa) CollectResults() {
}

type URLsFeFa struct {
	page       int
	urls       []string
	currURLidx int
	cb         FetchCallbacks
}

func (f *URLsFeFa) Prepare() {
	f.currURLidx = -1
	f.urls = f.cb.UrlsFetch(f.page)
}

func (f *URLsFeFa) Next() fefa.FetcherFast {
	f.currURLidx++
	if f.currURLidx < len(f.urls) {
		return &URLFeFa{url: f.urls[f.currURLidx], cb: f.cb}
	}
	return nil
}

func (f *URLsFeFa) CollectResults() {
}

type URLFeFa struct {
	url  string
	data interface{}
	cb   FetchCallbacks
}

func (f *URLFeFa) Prepare() {
	f.data = f.cb.UrlFetch(f.url)
}

func (f *URLFeFa) Next() fefa.FetcherFast {
	return nil
}

func (f *URLFeFa) CollectResults() {
	ResultsMu.Lock()
	Results = append(Results, f.data)
	ResultsMu.Unlock()
}
