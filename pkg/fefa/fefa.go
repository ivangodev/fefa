package fefa

import (
	"sync"
	"time"
)

type FetcherFast interface {
	Prepare()
	Next() FetcherFast
	CollectResults()
}

type Millisecond int
type RateLimitOpts struct {
	Interval Millisecond
	ReqsRate int
}

type rateLimiter struct {
	opts          *RateLimitOpts
	intervalStart time.Time
	currReqsCount int
	queue         chan interface{}
}

func newRateLimiter(r *RateLimitOpts) *rateLimiter {
	return &rateLimiter{opts: r, queue: make(chan interface{})}
}

func (r *rateLimiter) letRequest() bool {
	if r.intervalStart.IsZero() ||
		((time.Since(r.intervalStart) / time.Millisecond) >
			time.Duration(r.opts.Interval)) {
		r.intervalStart = time.Now()
		r.currReqsCount = 0
	}

	r.currReqsCount++
	return r.currReqsCount <= r.opts.ReqsRate
}

func (r *rateLimiter) queueController() {
	if r.opts == nil {
		return
	}

	for {
		tick := time.Tick(time.Duration(r.opts.Interval) * time.Millisecond / 10)
		select {
		case <-tick:
			if !r.letRequest() {
				break
			}

			_, ok := <-r.queue
			if !ok {
				return
			}

			if !r.letRequest() {
				break
			}

			for range r.queue {
				if !r.letRequest() {
					break
				}
			}
		}
	}
}

func (r *rateLimiter) closeQueueController() {
	close(r.queue)
}

func (r *rateLimiter) barrier() {
	if r.opts == nil {
		return
	}
	r.queue <- nil
}

func feFa(f FetcherFast, r *rateLimiter, parentGroup *sync.WaitGroup) {
	var waitGroup sync.WaitGroup

	f.Prepare()
	for na := f.Next(); na != nil; na = f.Next() {
		r.barrier()
		waitGroup.Add(1)
		go feFa(na, r, &waitGroup)
	}
	waitGroup.Wait()
	f.CollectResults()

	if parentGroup != nil {
		parentGroup.Done()
	}
}

func FeFa(f FetcherFast, r *RateLimitOpts) {
	rl := newRateLimiter(r)
	go rl.queueController()
	feFa(f, rl, nil)
	rl.closeQueueController()
}

func FeSlow(f FetcherFast) {
	f.Prepare()
	for na := f.Next(); na != nil; na = f.Next() {
		FeSlow(na)
	}
	f.CollectResults()
}
