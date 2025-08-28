package breaker

import (
	"errors"
	"sync"
	"time"
)

type Status int

const (
	Closed Status = iota
	Open
	HalfOpen
)

type Counter struct {
	Requests             uint64
	TotalSuccesses       uint64
	TotalFailures        uint64
	ConsecutiveSuccesses uint64
	ConsecutiveFailures  uint64
}

func (c *Counter) OnRequest() {
	c.Requests++
}

func (c *Counter) OnSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *Counter) OnFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

func (c *Counter) Reset() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

type Settings struct {
	Name           string
	MaxRequests    uint64
	Interval       time.Duration
	Timeout        time.Duration
	ReadyToTrip    func(counts *Counter) bool
	IsSuccess      func(err error) bool
	OnStatusChange func(name string, from, to Status)
}

type CircuitBreaker struct {
	name           string
	maxRequests    uint64
	interval       time.Duration
	timeout        time.Duration
	readyToTrip    func(counts *Counter) bool
	isSuccess      func(err error) bool
	onStatusChange func(name string, from, to Status)

	mutex      sync.Mutex
	status     Status
	generation uint64
	counter    *Counter
	expire     time.Time
}

func NewCircuitBreaker(settings Settings) *CircuitBreaker {
	if settings.Name == "" {
		settings.Name = "default"
	}
	if settings.MaxRequests == 0 {
		settings.MaxRequests = 1
	}
	if settings.Interval == 0 {
		settings.Interval = time.Duration(0)
	}
	if settings.Timeout == 0 {
		settings.Timeout = time.Duration(60) * time.Second
	}
	if settings.ReadyToTrip == nil {
		settings.ReadyToTrip = func(counts *Counter) bool {
			return counts.ConsecutiveFailures > 5
		}
	}
	if settings.IsSuccess == nil {
		settings.IsSuccess = func(err error) bool {
			return err == nil
		}
	}
	c := &CircuitBreaker{
		name:           settings.Name,
		maxRequests:    settings.MaxRequests,
		interval:       settings.Interval,
		timeout:        settings.Timeout,
		readyToTrip:    settings.ReadyToTrip,
		isSuccess:      settings.IsSuccess,
		onStatusChange: settings.OnStatusChange,
		status:         Closed,
		counter:        &Counter{},
	}
	c.NewGeneration()
	return c
}

func (c *CircuitBreaker) NewGeneration() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.generation++
	c.counter.Reset()
	switch c.status {
	case Open:
		c.expire = time.Now().Add(c.timeout)
	case HalfOpen:
		c.expire = time.Time{}
	case Closed:
		if c.interval > 0 {
			c.expire = time.Now().Add(c.interval)
		} else {
			c.expire = time.Time{}
		}
	}
}

func (c *CircuitBreaker) Execute(req func() (any, error)) (any, error) {
	gen, err := c.beforeRequest()
	if err != nil {
		return nil, err
	}
	resp, err := req()
	c.counter.OnRequest()
	c.afterRequest(gen, c.isSuccess(err))
	return resp, err
}

func (c *CircuitBreaker) beforeRequest() (uint64, error) {
	status, gen := c.currentStatus()
	switch status {
	case Open:
		return gen, errors.New("circuit breaker is open")
	case HalfOpen:
		if c.counter.Requests > c.maxRequests {
			return gen, errors.New("too many requests")
		}
	case Closed:
		return gen, nil
	}
	return gen, nil
}

func (c *CircuitBreaker) afterRequest(old uint64, success bool) {
	status, gen := c.currentStatus()
	if gen != old {
		return
	}
	if success {
		c.onSuccess(status)
	} else {
		c.onFailure(status)
	}
	return
}

func (c *CircuitBreaker) onSuccess(s Status) {
	switch s {
	case Closed:
		c.counter.OnSuccess()
	case HalfOpen:
		c.counter.OnSuccess()
		if c.counter.ConsecutiveSuccesses >= c.maxRequests {
			c.setStatus(Closed)
		}
	default:
		panic("unhandled default case")
	}
}

func (c *CircuitBreaker) onFailure(s Status) {
	switch s {
	case Closed:
		c.counter.OnFailure()
		if c.readyToTrip(c.counter) {
			c.setStatus(Open)
		}
	case HalfOpen:
		c.counter.OnFailure()
		if c.readyToTrip(c.counter) {
			c.setStatus(Open)
		}
	default:
		panic("unhandled default case")
	}
}

func (c *CircuitBreaker) currentStatus() (Status, uint64) {
	now := time.Now()
	switch c.status {
	case Closed:
		if !c.expire.IsZero() && c.expire.Before(now) {
			c.NewGeneration()
		}
	case Open:
		if c.expire.Before(now) {
			c.setStatus(HalfOpen)
		}
	default:
		panic("unhandled default case")
	}
	return c.status, c.generation
}

func (c *CircuitBreaker) setStatus(s Status) {
	if c.status == s {
		return
	}
	prev := c.status
	c.status = s
	c.NewGeneration()
	if c.onStatusChange != nil {
		c.onStatusChange(c.name, prev, s)
	}
}
