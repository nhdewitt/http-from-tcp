package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const inputFile = "messages.txt"

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
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Error opening %s", inputFile)
	}
	defer file.Close()

	lines := getLinesChannel(file)

	for line := range lines {
		fmt.Printf("read: %s\n", line)
	}
}
