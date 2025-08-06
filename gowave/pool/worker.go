package pool

import (
	gwlog "github.com/ChenGuo505/gowave/log"
	"time"
)

type Worker struct {
	pool    *Pool
	task    chan func()
	lastRun time.Time
}

func (w *Worker) Run() {
	go w.running()
}

func (w *Worker) running() {
	defer func() {
		w.pool.decRunning()
		w.pool.workerCache.Put(w)
		if r := recover(); r != nil {
			if w.pool.PanicHandler != nil {
				w.pool.PanicHandler()
			} else {
				gwlog.DefaultLogger().Error(r)
			}
		}
		w.pool.cond.Signal()
	}()
	for f := range w.task {
		if f == nil {
			w.pool.workerCache.Put(w)
			return
		}
		f()
		w.pool.putWorker(w)
		w.pool.decRunning()
	}
}
