package gowave

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

func Limiter(limit, cap int) MiddlewareFunc {
	l := rate.NewLimiter(rate.Limit(limit), cap)
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
			c, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := l.WaitN(c, 1)
			if err != nil {
				_ = ctx.String(http.StatusTooManyRequests, "too many requests")
				return
			}
			next(ctx)
		}
	}
}
