package response

import (
	"fmt"
	"io"
	"time"

	"github.com/nhdewitt/http-from-tcp/internal/headers"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	switch statusCode {
	case StatusOK:
		_, err := w.Write([]byte("HTTP/1.1 200 OK\r\n"))
		if err != nil {
			return err
		}
	case StatusBadRequest:
		_, err := w.Write([]byte("HTTP/1.1 400 Bad Request\r\n"))
		if err != nil {
			return err
		}
	case StatusInternalServerError:
		_, err := w.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n"))
		if err != nil {
			return err
		}
	default:
		_, err := w.Write([]byte(""))
		if err != nil {
			return err
		}
	}
	return nil
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()
	h.Set("Content-Length", fmt.Sprintf("%d", contentLen))
	h.Set("Connection", "close")
	h.Set("Content-Type", "text/plain")
	h.Set("Date", time.Now().UTC().Format(time.RFC1123))

	return h
}

func WriteHeaders(w io.Writer, headers headers.Headers) error {
	for k, v := range headers {
		caser := cases.Title(language.English)

		line := caser.String(k) + ": " + v
		_, err := w.Write([]byte(line + "\r\n"))
		if err != nil {
			return fmt.Errorf("error writing header: %v", err)
		}
	}
	_, err := w.Write([]byte("\r\n"))
	return err
}
