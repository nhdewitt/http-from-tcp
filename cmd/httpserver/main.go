package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nhdewitt/http-from-tcp/internal/request"
	"github.com/nhdewitt/http-from-tcp/internal/response"
	"github.com/nhdewitt/http-from-tcp/internal/server"
)

type htmlTemplate struct {
	status      []byte
	description []byte
	explanation []byte
}

const port = 42069

func Handler(w *response.Writer, req *request.Request) {
	var statusCode response.StatusCode
	switch req.RequestLine.RequestTarget {
	case "/yourproblem":
		statusCode = 400
	case "/myproblem":
		statusCode = 500
	default:
		statusCode = 200
	}
	err := w.WriteStatusLine(statusCode)
	if err != nil {
		return
	}

	var ht htmlTemplate
	switch statusCode {
	case 400:
		ht.status = []byte("400 Bad Request")
		ht.description = []byte("Bad Request")
		ht.explanation = []byte("Your request honestly kinda sucked.")
	case 500:
		ht.status = []byte("500 Internal Server Error")
		ht.description = []byte("Internal Server Error")
		ht.explanation = []byte("Okay, you know what? This one is on me.")
	default:
		ht.status = []byte("200 OK")
		ht.description = []byte("Success!")
		ht.explanation = []byte("Your request was an absolute banger.")
	}

	body := fmt.Appendf(nil, `
<html>
	<head>
		<title>%s</title>
	</head>
	<body>
		<h1>%s</h1>
		<p>%s</p>
	</body>
</html>
	`, ht.status, ht.description, ht.explanation)
	contentLength := len(body)

	h := response.GetDefaultHeaders(contentLength)
	h.SetNew("Content-Type", "text/html")
	w.WriteHeaders(h)
	n, err := w.WriteBody(body)
	if n != contentLength || err != nil {
		return
	}
}

func main() {
	server, err := server.Serve(port, Handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
