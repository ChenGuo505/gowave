package gowave

import (
	"log"
	"net"
	"strings"
	"time"
)

type LoggingConfig struct {
}

func LoggingWithConfig(conf LoggingConfig, next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		start := time.Now()
		path := ctx.Req.URL.Path
		raw := ctx.Req.URL.RawQuery
		method := ctx.Req.Method
		next(ctx)
		stop := time.Now()
		latency := stop.Sub(start)
		ip, _, _ := net.SplitHostPort(strings.TrimSpace(ctx.Req.RemoteAddr))
		clientIP := net.ParseIP(ip)
		statusCode := ctx.StatusCode
		if raw != "" {
			path = path + "?" + raw
		}
		log.Printf("[gowave] %v | %3d | %13v | %15s | %-7s %#v",
			stop.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	}
}

func Logging(next HandlerFunc) HandlerFunc {
	return LoggingWithConfig(LoggingConfig{}, next)
}
