package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

const port = ":42069"

func main() {
	addr, err := net.ResolveUDPAddr("udp", "localhost"+port)
	if err != nil {
		log.Fatalf("error resolving udp address: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("error preparing udp connection: %v", err)
	}
	defer conn.Close()

	r := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		line, err := r.ReadString('\n')
		if err != nil {
			log.Printf("input error: %v", err)
		}

		_, err = conn.Write([]byte(line))
		if err != nil {
			log.Printf("write error: %v", err)
		}
	}
}
