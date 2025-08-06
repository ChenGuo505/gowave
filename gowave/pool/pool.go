package pool

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const defaultExpire = 3

type Pool struct {
	cap     int32
	running int32
	workers []*Worker
	expire  time.Duration
	release chan sig
	lock    sync.Mutex
	once    sync.Once
}

type sig struct{}

func NewPool(cap int) *Pool {
	return NewPoolWithExpire(cap, defaultExpire)
}

func NewPoolWithExpire(cap int, expire int) *Pool {
	if cap <= 0 {
		return nil
	}
	if expire <= 0 {
		return nil
	}
	p := &Pool{
		cap:     int32(cap),
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
	}
	go p.cleanExpiredWorkers()
	return p
}

func (p *Pool) Submit(task func()) error {
	if len(p.release) > 0 {
		return errors.New("pool has been released")
	}
	w := p.getWorker()
	w.task <- task
	p.incRunning()
	return nil
}

func (p *Pool) Release() {
	p.once.Do(func() {
		p.lock.Lock()
		workers := p.workers
		for i, w := range workers {
			w.task = nil
			w.pool = nil
			workers[i] = nil
		}
		p.workers = nil
		p.lock.Unlock()
		p.release <- sig{}
	})
}

func (p *Pool) Restart() {
	if len(p.release) > 0 {
		return
	}
	<-p.release
	go p.cleanExpiredWorkers()
}

func (p *Pool) getWorker() *Worker {
	workers := p.workers
	idx := len(workers) - 1
	if idx >= 0 {
		p.lock.Lock()
		w := workers[idx]
		workers[idx] = nil
		p.workers = workers[:idx]
		p.lock.Unlock()
		return w
	}
	if p.running < p.cap {
		w := &Worker{
			pool: p,
			task: make(chan func(), 1),
		}
		w.Run()
		return w
	}
	for {
		workers := p.workers
		idx := len(workers) - 1
		if idx < 0 {
			continue
		}
		p.lock.Lock()
		w := workers[idx]
		workers[idx] = nil
		p.workers = workers[:idx]
		p.lock.Unlock()
		return w
	}
}

func (p *Pool) putWorker(w *Worker) {
	w.lastRun = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, w)
	p.lock.Unlock()
}

func (p *Pool) incRunning() {
	atomic.AddInt32(&p.running, 1)
}

func (p *Pool) decRunning() {
	atomic.AddInt32(&p.running, -1)
}

func (p *Pool) cleanExpiredWorkers() {
	ticker := time.NewTicker(p.expire)
	for range ticker.C {
		if len(p.release) > 0 {
			return
		}
		p.lock.Lock()
		workers := p.workers
		idx := -1
		for i, w := range workers {
			if time.Now().Sub(w.lastRun) <= p.expire {
				break
			}
			idx = i
			w.task <- nil
			workers[i] = nil
		}
		if idx >= 0 {
			if idx >= len(workers)-1 {
				p.workers = workers[:0]
			} else {
				p.workers = workers[idx+1:]
			}
		}
		p.lock.Unlock()
	}
}
