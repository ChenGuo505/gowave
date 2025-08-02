package main

import (
	"fmt"
	"github.com/ChenGuo505/gowave"
	"log"
	"net/http"
)

type User struct {
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age" validate:"gte=0,lte=130"`
}

func main() {
	engine := gowave.New()
	g := engine.Group("api")
	g.Use(gowave.Logging)
	g.Get("/hello", func(ctx *gowave.Context) {
		fmt.Println("Hello Handler")
		_, err := fmt.Fprintf(ctx.W, "Hello World")
		if err != nil {
			return
		}
	}, func(next gowave.HandlerFunc) gowave.HandlerFunc {
		return func(ctx *gowave.Context) {
			fmt.Println("Middleware 1")
			next(ctx)
			fmt.Println("Middleware 1 End")
		}
	})
	g.Post("/test", func(ctx *gowave.Context) {
		_, err := fmt.Fprintf(ctx.W, "Post Test")
		if err != nil {
			return
		}
	})
	g.Get("/user/:id", func(ctx *gowave.Context) {
		_, err := fmt.Fprintf(ctx.W, "User Test")
		if err != nil {
			return
		}
	})
	g.Get("/order/*/info", func(ctx *gowave.Context) {
		_, err := fmt.Fprintf(ctx.W, "Order Info Test")
		if err != nil {
			return
		}
	})
	g.Get("/html", func(ctx *gowave.Context) {
		err := ctx.HTML(http.StatusOK, "<h1>Hello World</h1>")
		if err != nil {
			log.Fatal(err)
			return
		}
	})
	engine.LoadTemplate("templates/*.html")
	g.Get("/template", func(ctx *gowave.Context) {
		user := User{Name: "John Doe"}
		err := ctx.Template("login.html", user)
		if err != nil {
			log.Fatal(err)
			return
		}
	})
	g.Get("/json", func(ctx *gowave.Context) {
		user := User{Name: "John Doe"}
		err := ctx.JSON(http.StatusOK, user)
		if err != nil {
			log.Fatal(err)
			return
		}
	})
	g.Get("/xml", func(ctx *gowave.Context) {
		user := User{Name: "John Doe"}
		err := ctx.XML(http.StatusOK, user)
		if err != nil {
			log.Fatal(err)
			return
		}
	})
	g.Post("/json", func(ctx *gowave.Context) {
		users := make([]User, 0)
		ctx.DisallowUnknownFields = true
		ctx.EnableJsonValidation = true
		err := ctx.BindJson(&users)
		if err == nil {
			err := ctx.JSON(http.StatusOK, users)
			if err != nil {
				return
			}
		} else {
			log.Println(err)
		}
	})
	engine.Run()
}
