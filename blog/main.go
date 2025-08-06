package main

import (
	"fmt"
	"github.com/ChenGuo505/gowave"
	"github.com/ChenGuo505/gowave/pool"
	"net/http"
	"sync"
	"time"
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
	g.Get("/task", func(ctx *gowave.Context) {
		p := pool.NewPool(3)
		var wg sync.WaitGroup
		wg.Add(3)
		_ = p.Submit(func() {
			fmt.Println("1111")
			time.Sleep(3 * time.Second)
			wg.Done()
		})
		_ = p.Submit(func() {
			fmt.Println("2222")
			time.Sleep(3 * time.Second)
			wg.Done()
		})
		_ = p.Submit(func() {
			fmt.Println("3333")
			time.Sleep(3 * time.Second)
			wg.Done()
		})
		wg.Wait()
		err := ctx.String(http.StatusOK, "tasks completed")
		if err != nil {
			ctx.Logger.Error(err)
			return
		}
	})
	engine.Run()
}
