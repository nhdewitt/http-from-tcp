package main

import (
	"fmt"
	"log"
	"net"

	"github.com/nhdewitt/http-from-tcp/internal/request"
)

const port = ":42069"

func main() {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("error listening: %v", err.Error())
	}
	defer listener.Close()

	fmt.Println("Listening for TCP traffic on", port)
	for {
		c, err := listener.Accept()
		if err != nil {
			log.Fatalf("error accepting connection: %v", err)
		}
		log.Println("Connection accepted:", c.RemoteAddr())

		req, err := request.RequestFromReader(c)
		if err != nil {
			log.Fatalf("error parsing request: %v", err)
		}

		fmt.Println("Request line:")
		fmt.Printf("- Method: %s\n", req.RequestLine.Method)
		fmt.Printf("- Target: %s\n", req.RequestLine.RequestTarget)
		fmt.Printf("- Version: %s\n", req.RequestLine.HttpVersion)
		fmt.Println("Headers:")
		for k, v := range req.Headers {
			fmt.Printf("- %s: %s\n", k, v)
		}
		body := string(req.Body)
		fmt.Println("Body:")
		fmt.Println(body)
		fmt.Println("Connection to ", c.RemoteAddr(), "closed")
	}
}
