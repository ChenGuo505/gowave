package gowave

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

func errorDetails(err any) string {
	var pcs [32]uintptr
	n := runtime.Callers(3, pcs[:])
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%v\n", err))
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		sb.WriteString(fmt.Sprintf("\t %s:%d %s\n", file, line, fn.Name()))
	}
	return sb.String()
}

func Recovery(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				ctx.Logger.Error(errorDetails(err))
				ctx.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		next(ctx)
	}
}
