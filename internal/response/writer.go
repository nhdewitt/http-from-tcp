package response

import (
	"errors"
	"fmt"
	"io"

	"github.com/nhdewitt/http-from-tcp/internal/headers"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type writerState int

const (
	StateWritingStatusLine writerState = iota
	StateWritingHeaders
	StateWritingBody
	StateDone
)

type Writer struct {
	writer io.Writer
	state  writerState
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer: w,
		state:  StateWritingStatusLine,
	}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.state != StateWritingStatusLine {
		return fmt.Errorf("writer state out-of-order")
	}

	switch statusCode {
	case StatusOK:
		w.writer.Write([]byte("HTTP/1.1 200 OK\r\n"))
	case StatusBadRequest:
		w.writer.Write([]byte("HTTP/1.1 400 Bad Request\r\n"))
	case StatusInternalServerError:
		w.writer.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n"))
	default:
		w.writer.Write([]byte(""))
	}

	w.state = StateWritingHeaders
	return nil
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	if w.state != StateWritingHeaders {
		return fmt.Errorf("writer state out-of-order")
	}

	caser := cases.Title(language.English)
	for k, v := range headers {
		line := caser.String(k) + ": " + v
		_, err := w.writer.Write([]byte(line + "\r\n"))
		if err != nil {
			return errors.New("error writing headers")
		}
	}
	w.writer.Write([]byte("\r\n"))

	w.state = StateWritingBody
	return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.state != StateWritingBody {
		return 0, fmt.Errorf("writer state out-of-order")
	}

	w.state = StateDone
	return w.writer.Write(p)
}
