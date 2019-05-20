package main

import (
	"flag"
	"fmt"
	"github.com/liukaijv/win"
	"log"
	"net/http"
)

type HelloResponse struct {
	Name string
}

func Hello(c win.Context) {
	var helloRequest struct {
		Name string
	}
	err := c.Bind(&helloRequest)
	if err != nil {
		c.ReplyError(200, err.Error())
	}
	c.Reply(helloRequest)
}

func logMiddleware(next win.HandlerFunc) win.HandlerFunc {
	return func(ctx win.Context) {
		fmt.Println("request id:", ctx.Request.ID)
		next(ctx)
	}
}

func main() {

	var addr = flag.String("addr", ":8080", "http service address")

	s := win.NewServer()

	s.Use(logMiddleware)

	s.AddHandler("hello", Hello)

	http.HandleFunc("/ws", s.Serve)

	log.Println("server addr: " + *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
