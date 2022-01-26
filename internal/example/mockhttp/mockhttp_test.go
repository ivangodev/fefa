package mockhttp

import (
	"github.com/ivangodev/fefa/internal/example"
	"github.com/ivangodev/fefa/pkg/fefa"
	"github.com/ivangodev/fefa/entity"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"
)

const (
	pagesNumber = 3
	urlsPerPage = 10
	baseURL     = "example.com"
)

func defaultSleep() {
	time.Sleep(10 * time.Millisecond)
}

func pagesFetch(page int) (validPage bool) {
	validPage = page > 0 && page <= pagesNumber
	defaultSleep()
	return
}

func getURL(page, urlNum int) string {
	return baseURL + strconv.Itoa(page*urlsPerPage+urlNum)
}

func urlsFetch(page int) (URLs []string) {
	defaultSleep()
	URLs = make([]string, 0)
	for i := 0; i < urlsPerPage; i++ {
		URLs = append(URLs, getURL(page, i))
	}
	defaultSleep()
	return
}

func getData(URL string) string {
	return URL + "data"
}

func urlFetch(URL string) (data interface{}) {
	data = getData(URL)
	defaultSleep()
	return
}

func getExpectedData() []string {
	res := make([]string, 0)
	for p := 1; p <= pagesNumber; p++ {
		for u := 0; u < urlsPerPage; u++ {
			res = append(res, getData(getURL(p, u)))
		}
	}
	return res
}

func getActualData() []string {
	res := make([]string, len(example.Results))
	for i, v := range example.Results {
		res[i] = v.(string)
	}
	sort.Strings(res)
	return res
}

func compareResults(t *testing.T) {
	expected := getExpectedData()
	actual := getActualData()

	expectedLen := len(expected)
	actualLen := len(actual)
	if expectedLen != actualLen {
		t.Fatalf("Unexpected length of results: want %v VS actual %v; (%v VS %v)",
			expectedLen, actualLen, expected, actual)
	}

	for i, w := range expected {
		if a := actual[i]; w != a {
			t.Fatalf("Unexpected value in results at index %v: want %v VS actual %v; (%v VS %v)",
				i, w, a, expected, actual)
		}
	}
}

type rateLimiter struct {
	t            *testing.T
	opts         fefa.RateLimitOpts
	reqCnt       int
	mu           sync.Mutex
	rateViolated bool
}

func newRateLimiter(t *testing.T, opts fefa.RateLimitOpts) *rateLimiter {
	return &rateLimiter{t: t, opts: opts}
}

func (rl *rateLimiter) registerReq() {
	rl.mu.Lock()
	rl.reqCnt++
	rl.mu.Unlock()
}

func (rl *rateLimiter) controller() {
	for {
		rl.mu.Lock()
		rl.reqCnt = 0
		rl.mu.Unlock()
		tick := time.Tick(time.Duration(rl.opts.Interval) * time.Millisecond)
		select {
		case <-tick:
			rl.mu.Lock()
			cnt := rl.reqCnt
			rl.mu.Unlock()
			if cnt > rl.opts.ReqsRate {
				rl.rateViolated = true
				rl.t.Logf("Rate violated: actual %v VS want %v",
					cnt, rl.opts.ReqsRate)
				return
			}
		}
	}
}

func (rl *rateLimiter) limitCallbacks(cb example.FetchCallbacks) (cbLim example.FetchCallbacks) {
	cbLim = example.FetchCallbacks{
		PagesFetch: func(page int) (validPage bool) {
			rl.registerReq()
			validPage = cb.PagesFetch(page)
			return
		},
		UrlsFetch: func(page int) (URLs []string) {
			rl.registerReq()
			URLs = cb.UrlsFetch(page)
			return
		},
		UrlFetch: func(URL string) (data interface{}) {
			rl.registerReq()
			data = cb.UrlFetch(URL)
			return
		},
	}
	return
}

func TestFeFa(t *testing.T) {
	cb := example.FetchCallbacks{pagesFetch, urlsFetch, urlFetch}
	fefa.FeFa(&example.PagesFeFa{Cb: cb}, nil)
	compareResults(t)
}

func testFeFaRateLimit(t *testing.T, ctrlOpts, fetcherOpts fefa.RateLimitOpts) *rateLimiter {
	rl := newRateLimiter(t, ctrlOpts)
	go rl.controller()
	cb := rl.limitCallbacks(example.FetchCallbacks{pagesFetch, urlsFetch, urlFetch})
	fefa.FeFa(&example.PagesFeFa{Cb: cb}, &fetcherOpts)
	compareResults(t)
	return rl
}

func TestFeFaWithinRateLimit(t *testing.T) {
	ctrlOpts := fefa.RateLimitOpts{Interval: 10, ReqsRate: 1}
	fetcherOpts := ctrlOpts
	_ = testFeFaRateLimit(t, ctrlOpts, fetcherOpts)
}

func TestFeFaExceedRateLimit(t *testing.T) {
	ctrlOpts := fefa.RateLimitOpts{Interval: 10, ReqsRate: 1}
	fetcherOpts := ctrlOpts
	fetcherOpts.ReqsRate *= 2
	rl := testFeFaRateLimit(t, ctrlOpts, fetcherOpts)
	if !rl.rateViolated {
		t.Fatalf("Rate expected to be violated")
	}
}

func TestFeSlow(t *testing.T) {
	cb := example.FetchCallbacks{pagesFetch, urlsFetch, urlFetch}
	fefa.FeSlow(&example.PagesFeFa{Cb: cb})
	compareResults(t)
}
