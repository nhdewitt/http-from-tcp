package headers

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	crlf                = "\r\n"
	validFieldNameChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%&'*+-.^_`|~"
)

type Headers map[string]string

func NewHeaders() Headers {
	return map[string]string{}
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	idx := bytes.Index(data, []byte(crlf))
	if idx == -1 {
		return 0, false, nil
	}
	if idx == 0 {
		n = idx + 2
		return n, true, nil
	}

	fields := data[:idx]
	colonIdx := bytes.IndexByte(fields, ':')
	if colonIdx == -1 {
		return 0, false, fmt.Errorf("malformed header line (no colon): %q", fields)
	}

	prefix := fields[:colonIdx]
	name := bytes.TrimLeft(prefix, " \t")

	if len(name) == 0 || bytes.ContainsAny(prefix, " \t") {
		return 0, false, fmt.Errorf("malformed field-name: %q", fields)
	}

	pr := bytes.TrimRight(prefix, " \t")

	if len(pr) != len(prefix) {
		return 0, false, fmt.Errorf("malformed field-name (space before colon): %q", fields)
	}
	if !bytes.HasSuffix(pr, name) {
		return 0, false, fmt.Errorf("malformed field-name: %q", fields)
	}

	key := string(name)
	value := string(bytes.TrimSpace(fields[colonIdx+1:]))
	n = idx + 2

	for _, r := range key {
		if !strings.Contains(validFieldNameChars, string(r)) {
			return 0, false, fmt.Errorf("invalid character in field-name: %q", fields)
		}
	}

	h.Set(key, value)

	return n, false, nil
}

func (h Headers) Set(key, value string) {
	key = strings.ToLower(key)
	if _, ok := h[strings.ToLower(key)]; ok {
		h[key] += ", " + value
		return
	}
	h[key] = value
}

func (h Headers) SetNew(key, value string) {
	key = strings.ToLower(key)
	h[key] = value
}

func (h Headers) Get(key string) (value string) {
	key = strings.ToLower(key)
	if v, ok := h[strings.ToLower(key)]; ok {
		return v
	}
	return ""
}

func (h Headers) Del(key string) {
	key = strings.ToLower(key)
	delete(h, key)
}
