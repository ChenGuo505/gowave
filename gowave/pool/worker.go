package pool

import "time"

type Worker struct {
	pool    *Pool
	task    chan func()
	lastRun time.Time
}

func (w *Worker) Run() {
	go w.running()
}

func (w *Worker) running() {
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
