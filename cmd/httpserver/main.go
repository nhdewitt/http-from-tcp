package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

const (
	buffer   = 1024
	port     = 42069
	upstream = "https://httpbin.org/"
)

func Handler(w *response.Writer, req *request.Request) {
	var statusCode response.StatusCode
	target := req.RequestLine.RequestTarget
	if strings.HasPrefix(target, "/httpbin") {
		proxy(target, w)
		return
	} else if strings.HasPrefix(target, "/video") {
		serveVideo(w)
		return
	}
	switch target {
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
	bodyBytes := len(body)

	h := response.GetDefaultHeaders(bodyBytes)
	h.SetNew("Content-Type", "text/html")
	w.WriteHeaders(h)
	n, err := w.WriteBody(body)
	if n != bodyBytes || err != nil {
		return
	}
}

func serveVideo(w *response.Writer) {
	video, err := os.ReadFile("assets/vim.mp4")
	if err != nil {
		return
	}

	w.WriteStatusLine(200)
	h := response.GetDefaultHeaders(len(video))
	h.SetNew("Content-Type", "video/mp4")
	if err := w.WriteHeaders(h); err != nil {
		return
	}
	if _, err := w.WriteBody(video); err != nil {
		return
	}
}

func proxy(target string, w *response.Writer) {
	target = strings.TrimPrefix(target, "/httpbin")
	if !strings.HasPrefix(target, "/") {
		target = "/" + target
	}
	url := upstream + target

	var resp *http.Response
	resp, err := http.Get(url)
	if err != nil {
		_ = w.WriteStatusLine(502)
		return
	}
	defer resp.Body.Close()

	if err = w.WriteStatusLine(response.StatusCode(resp.StatusCode)); err != nil {
		return
	}

	h := response.GetDefaultHeaders(0)
	h.SetNew("Content-Type", resp.Header.Get("Content-Type"))
	h.Del("Content-Length")
	h.SetNew("Transfer-Encoding", "chunked")
	h.SetNew("Trailer", "X-Content-SHA256, X-Content-Length")
	if err := w.WriteHeaders(h); err != nil {
		return
	}

	b := make([]byte, buffer)
	var bodyBytes []byte
	for {
		n, rerr := resp.Body.Read(b)
		if n > 0 {
			if _, err := w.WriteChunkedBody(b[:n]); err != nil {
				return
			}
			bodyBytes = append(bodyBytes, b[:n]...)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			if len(bodyBytes) != 0 {
				break
			}
			return
		}
	}
	h.SetNew("X-Content-Length", fmt.Sprintf("%d", len(bodyBytes)))
	h.SetNew("X-Content-SHA256", fmt.Sprintf("%x", sha256.Sum256(bodyBytes)))
	if _, err = w.WriteChunkedBodyDone(h); err != nil {
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
