package main

import (
	"fmt"
	"github.com/liukaijv/win"
	"log"
)

func main() {

	var urlStr = "ws://localhost:8080/ws"
	client, err := win.Dial(urlStr, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	for i := 1; i < 10000; i++ {
		go func() {

			var reply struct {
				Name string
			}

			helloRequest := struct {
				Name string
			}{
				Name: "haha",
			}

			err = client.Call("hello", helloRequest, &reply)

			if err != nil {
				fmt.Printf("err %v\n", err)
				return
			}

			fmt.Printf("response %+v\n", reply)
		}()
	}

	select {}

}
