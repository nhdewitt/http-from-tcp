package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

const port = ":42069"

func getLinesChannel(f io.ReadCloser) <-chan string {
	out := make(chan string)

	currentLine := ""
	go func() {
		defer close(out)
		defer f.Close()

		buf := make([]byte, 8)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				parts := strings.Split(string(buf[:n]), "\n")
				for i := 0; i < len(parts)-1; i++ {
					out <- strings.TrimSuffix(currentLine+parts[i], "\r")
					currentLine = ""
				}
				currentLine += parts[len(parts)-1]
			}
			if errors.Is(err, io.EOF) {
				if currentLine != "" {
					out <- strings.TrimSuffix(currentLine, "\r")
					currentLine = ""
				}
				return
			}
			if err != nil {
				return
			}
		}
	}()

	return out
}

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

		lines := getLinesChannel(c)

		for line := range lines {
			fmt.Println(line)
		}
		fmt.Println("Connection to ", c.RemoteAddr(), "closed")
	}
}
