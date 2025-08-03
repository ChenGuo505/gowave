package gowave

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	greenBg   = "\033[97;42m"
	whiteBg   = "\033[90;47m"
	yellowBg  = "\033[90;43m"
	redBg     = "\033[97;41m"
	blueBg    = "\033[97;44m"
	magentaBg = "\033[97;45m"
	cyanBg    = "\033[97;46m"
	green     = "\033[32m"
	white     = "\033[37m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
	reset     = "\033[0m"
)

var DefaultWriter io.Writer = os.Stdout

type LoggingConfig struct {
	Formatter LogFormatter // Custom log formatter
	out       io.Writer    // Output writer, defaults to os.Stdout
}

type LogFormatter func(params *LogFormatterParams) string

type LogFormatterParams struct {
	Request    *http.Request
	Timestamp  time.Time
	StatusCode int
	Latency    time.Duration
	ClientIP   net.IP
	Method     string
	Path       string

	IsColored bool // Whether to use colored output
}

func (p *LogFormatterParams) StatusCodeColor() string {
	code := p.StatusCode
	switch code {
	case http.StatusOK:
		return green
	default:
		return red
	}
}

var defaultLogFormatter = func(params *LogFormatterParams) string {
	statusCodeColor := params.StatusCodeColor()
	if params.Latency > time.Minute {
		params.Latency = params.Latency.Truncate(time.Second)
	}
	if params.IsColored {
		return fmt.Sprintf("%s[gowave]%s |%s %v %s|%s %3d %s|%s %13v %s| %15s |%s %-7s %s %s %#v %s\n",
			cyan, reset,
			blue, params.Timestamp.Format("2006/01/02 15:04:05"), reset,
			statusCodeColor, params.StatusCode, reset,
			magenta, params.Latency, reset,
			params.ClientIP,
			magenta, params.Method, reset,
			cyan, params.Path, reset,
		)
	}
	return fmt.Sprintf("[gowave] | %v | %3d | %13v | %15s | %-7s %#v\n",
		params.Timestamp.Format("2006/01/02 15:04:05"),
		params.StatusCode,
		params.Latency,
		params.ClientIP,
		params.Method,
		params.Path,
	)
}

func LoggingWithConfig(conf LoggingConfig, next HandlerFunc) HandlerFunc {
	formatter := conf.Formatter
	if formatter == nil {
		formatter = defaultLogFormatter
	}
	out := conf.out
	if out == nil {
		out = DefaultWriter
	}
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
		params := &LogFormatterParams{
			Request:    ctx.Req,
			Timestamp:  stop,
			StatusCode: statusCode,
			Latency:    latency,
			ClientIP:   clientIP,
			Method:     method,
			Path:       path,
			IsColored:  true,
		}
		_, err := fmt.Fprintf(out, formatter(params))
		if err != nil {
			return
		}
	}
}

func Logging(next HandlerFunc) HandlerFunc {
	return LoggingWithConfig(LoggingConfig{}, next)
}
