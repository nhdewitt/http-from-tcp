package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

type requestState int

const (
	bufferSize                    = 8
	crlf                          = "\r\n"
	stateInitialized requestState = iota
	stateDone
)

type Request struct {
	RequestLine RequestLine
	state       requestState
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	buf := make([]byte, bufferSize)
	readToIndex := 0

	r := Request{
		state: stateInitialized,
	}

	for r.state == stateInitialized {
		if readToIndex == len(buf) {
			tmpBuf := make([]byte, len(buf)*2)
			copy(tmpBuf, buf[:readToIndex])
			buf = tmpBuf
		}

		n, err := reader.Read(buf[readToIndex:])
		if n > 0 {
			readToIndex += n

			bytesParsed, perr := r.parse(buf[:readToIndex])
			if perr != nil {
				return nil, perr
			}

			copy(buf, buf[bytesParsed:readToIndex])
			readToIndex -= bytesParsed
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				if r.state != stateDone {
					return nil, fmt.Errorf("error parsing data: early EOF")
				}
				break
			}
			return nil, err
		}
	}

	return &r, nil
}

func (r *Request) parse(data []byte) (int, error) {
	switch r.state {
	case stateInitialized:
		parsed, parsedRequest, err := parseRequestLine(data)
		if parsed == 0 && err == nil {
			return 0, nil
		}
		if err != nil {
			return 0, fmt.Errorf("error parsing data: %v", err)
		}

		r.RequestLine = parsedRequest
		r.state = stateDone

		return parsed, nil
	case stateDone:
		return 0, fmt.Errorf("error: trying to read data in a done state")
	default:
		return 0, fmt.Errorf("error: unknown state")
	}
}

func parseRequestLine(req []byte) (int, RequestLine, error) {
	idx := bytes.Index(req, []byte(crlf))
	if idx == -1 {
		return 0, RequestLine{}, nil
	}
	line := string(req[:idx])
	consumed := idx + len(crlf)

	rl, err := requestLineFromString(line)
	if err != nil {
		return 0, RequestLine{}, err
	}

	return consumed, *rl, nil
}

func requestLineFromString(s string) (*RequestLine, error) {
	parts := strings.Fields(s)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid request line: %s", s)
	}

	method := parts[0]
	for _, c := range method {
		if c < 'A' || c > 'Z' {
			return nil, fmt.Errorf("invalid method: %s", method)
		}
	}

	target := parts[1]

	protocol, version, ok := strings.Cut(parts[2], "/")
	if !ok || protocol != "HTTP" {
		return nil, fmt.Errorf("invalid HTTP version: %s", parts[2])
	}
	if version != "1.1" {
		return nil, fmt.Errorf("invalid HTTP version: %s", parts[2])
	}

	return &RequestLine{
		Method:        method,
		RequestTarget: target,
		HttpVersion:   version,
	}, nil
}
