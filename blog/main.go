package main

import (
	"github.com/ChenGuo505/gowave"
	"net/http"
)

func main() {
	engine := gowave.New()
	g := engine.Group("api")
	g.Get("/hello", func(ctx *gowave.Context) {
		ctx.Logger.Info("hello world!")
		err := ctx.String(http.StatusOK, "hello world!")
		if err != nil {
			ctx.Logger.Error(err)
			return
		}
	})
	engine.Run()
}
