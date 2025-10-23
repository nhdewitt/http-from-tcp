package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func NewHeaders() Headers {
	return make(Headers)
}

func TestRequestLineParse(t *testing.T) {
	// Test: Valid single header
	headers := NewHeaders()
	data := []byte("Host: localhost:42069\r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, 23, n)
	assert.False(t, done)
	n, done, err = headers.Parse(data[n:])
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, 2, n)
	assert.True(t, done)

	// Test: Valid header with leading whitespace
	headers = NewHeaders()
	data = []byte(" Host: localhost:42069 \r\n")
	_, _, err = headers.Parse(data)
	require.Error(t, err)

	// Test: Invalid spacing header
	headers = NewHeaders()
	data = []byte("           Host : localhost:42069             \r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: Valid 3 headers
	headers = NewHeaders()
	data = []byte("Host: example.com\r\nUser-Agent: test-agent/1.0\r\nAccept: */*\r\n\r\n")
	n, done, err = headers.Parse(data)
	data = data[n:]
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "example.com", headers["host"])
	assert.Equal(t, 19, n)
	assert.False(t, done)
	n, done, err = headers.Parse(data)
	data = data[n:]
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "test-agent/1.0", headers["user-agent"])
	assert.Equal(t, 28, n)
	assert.False(t, done)
	n, done, err = headers.Parse(data)
	data = data[n:]
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, "*/*", headers["accept"])
	assert.Equal(t, 13, n)
	assert.False(t, done)
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, 2, n)
	assert.True(t, done)

	// Valid done
	headers = NewHeaders()
	data = []byte("\r\n extra text ignored")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.True(t, done)

	// Partial line (no CRLF)
	headers = NewHeaders()
	data = []byte("Host: loca")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Valid 2 headers with existing headers
	headers = NewHeaders()
	headers["user-agent"] = "curl/7.54.1"
	headers["accept-language"] = "en-US"
	data = []byte("Host: localhost:42069\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, 23, n)
	assert.False(t, done)
	assert.Equal(t, "localhost:42069", headers["host"])
	assert.Equal(t, "en-US", headers["accept-language"])
	assert.Equal(t, "curl/7.54.1", headers["user-agent"])

	// Invalid no colon
	headers = NewHeaders()
	data = []byte("Host localhost:42069\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)
	headers = NewHeaders()
	data = []byte("Host localhost 42069\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Invalid character in header key
	headers = NewHeaders()
	data = []byte("H©st: localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Multiple values for one header key
	headers = NewHeaders()
	data = []byte("Set-Person: lane-loves-go\r\nSet-Person: prime-loves-zig\r\nSet-Person: tj-loves-ocaml\r\n\r\n")
	for range 4 {
		n, done, err = headers.Parse(data)
		data = data[n:]
	}
	require.NoError(t, err)
	assert.Equal(t, "lane-loves-go, prime-loves-zig, tj-loves-ocaml", headers["set-person"])
	assert.True(t, done)
}
