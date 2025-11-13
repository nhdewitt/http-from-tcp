package response

import (
	"fmt"
	"io"
	"strings"

	"github.com/nhdewitt/http-from-tcp/internal/headers"
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

	var line string
	switch statusCode {
	case StatusOK:
		line = "HTTP/1.1 200 OK\r\n"
	case StatusBadRequest:
		line = "HTTP/1.1 400 Bad Request\r\n"
	case StatusInternalServerError:
		line = "HTTP/1.1 500 Internal Server Error\r\n"
	default:
		return fmt.Errorf("unknown status code: %d", statusCode)
	}
	if _, err := w.writer.Write([]byte(line)); err != nil {
		return err
	}

	w.state = StateWritingHeaders
	return nil
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	if w.state != StateWritingHeaders {
		return fmt.Errorf("writer state out-of-order")
	}

	for k, v := range headers {
		if v == "" {
			continue
		}
		h := k + ": " + v + "\r\n"
		if _, err := w.writer.Write([]byte(h)); err != nil {
			return err
		}
	}
	if _, err := w.writer.Write([]byte("\r\n")); err != nil {
		return err
	}

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

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if w.state != StateWritingBody {
		return 0, fmt.Errorf("writer state out-of-order")
	}

	n := len(p)
	if n == 0 {
		return 0, nil
	}

	if _, err := w.writer.Write([]byte(fmt.Sprintf("%X\r\n", n))); err != nil {
		return 0, err
	}
	if _, err := w.writer.Write(p); err != nil {
		return 0, err
	}
	if _, err := w.writer.Write([]byte("\r\n")); err != nil {
		return 0, err
	}
	return n, nil
}

func (w *Writer) WriteChunkedBodyDone(h headers.Headers) (int, error) {
	if w.state != StateWritingBody {
		return 0, fmt.Errorf("writer state out-of-order")
	}
	if _, err := w.writer.Write([]byte("0\r\n")); err != nil {
		return 0, err
	}
	if err := w.WriteTrailers(h); err != nil {
		return 0, err
	}
	if _, err := w.writer.Write([]byte("\r\n")); err != nil {
		return 0, err
	}
	w.state = StateDone
	return 0, nil
}

func (w *Writer) WriteTrailers(h headers.Headers) error {
	t := h.Get("Trailer")
	if len(t) == 0 {
		return nil
	}

	for k := range strings.SplitSeq(t, ",") {
		k = strings.TrimSpace(k)
		if len(k) == 0 {
			continue
		}
		v := h.Get(k)
		if len(v) == 0 {
			continue
		}
		if _, err := w.writer.Write([]byte(k + ": " + v + "\r\n")); err != nil {
			return err
		}
	}
	return nil
}
