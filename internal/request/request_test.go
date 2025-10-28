package request

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

// Read reads up to len(p) or numBytesPerRead bytes from the string per cell
// its useful for simulating reading a variable number of bytes per chunk from a network connection
func (cr *chunkReader) Read(p []byte) (n int, err error) {
	if cr.pos >= len(cr.data) {
		return 0, io.EOF
	}
	endIndex := cr.pos + cr.numBytesPerRead
	if endIndex > len(cr.data) {
		endIndex = len(cr.data)
	}
	n = copy(p, cr.data[cr.pos:endIndex])
	cr.pos += n

	return n, nil
}

func TestRequestLineParse(t *testing.T) {
	cases := []struct {
		data                            string
		wantMethod, wantTarget, wantVer string
	}{
		{"GET / HTTP/1.1\r\nHost: x\r\n\r\n", "GET", "/", "1.1"},
		{"GET /coffee HTTP/1.1\r\nHost: x\r\n\r\n", "GET", "/coffee", "1.1"},
	}
	for _, c := range cases {
		for _, chunk := range []int{1, 2, 3, len(c.data)} {
			reader := &chunkReader{data: c.data, numBytesPerRead: chunk}
			r, err := RequestFromReader(reader)
			require.NoError(t, err)
			require.NotNil(t, r)
			assert.Equal(t, c.wantMethod, r.RequestLine.Method)
			assert.Equal(t, c.wantTarget, r.RequestLine.RequestTarget)
			assert.Equal(t, c.wantVer, r.RequestLine.HttpVersion)
		}
	}

	// Test: Invalid request line: missing method
	bad := "/coffee HTTP/1.1\r\nHost: x\r\n\r\n"
	reader := &chunkReader{data: bad, numBytesPerRead: len(bad)}
	_, err := RequestFromReader(reader)
	require.Error(t, err)

	// Invalid method order
	bad = "/coffee GET HTTP/1.1\r\nHost: x\r\n\r\n"
	reader = &chunkReader{data: bad, numBytesPerRead: len(bad)}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Invalid version
	bad = "GET /coffee HTTP/3.0\r\nHost: x\r\n\r\n"
	reader = &chunkReader{data: bad, numBytesPerRead: len(bad)}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Early EOF (no CRLF)
	bad = "GET /coffee HTTP/1.1"
	reader = &chunkReader{data: bad, numBytesPerRead: 2}
	_, err = RequestFromReader(reader)
	require.Error(t, err)
}

func TestFullSetOfHeaders(t *testing.T) {
	// Test: Standard Headers
	reader := &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
		numBytesPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "localhost:42069", r.Headers["host"])
	assert.Equal(t, "curl/7.81.0", r.Headers["user-agent"])
	assert.Equal(t, "*/*", r.Headers["accept"])

	// Test: Malformed Header
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost localhost:42069\r\n\r\n",
		numBytesPerRead: 3,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Empty Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\n\r\n",
		numBytesPerRead: 1,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	assert.Empty(t, r.Headers)
	assert.Equal(t, stateDone, r.state)

	// Test: Case-Insensitive Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nhOsT: localhost:42069\r\nUSER-AGENT: curl/7.81.0\r\n\r\n",
		numBytesPerRead: 10,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	assert.Equal(t, "localhost:42069", r.Headers["host"])
	assert.Equal(t, "curl/7.81.0", r.Headers["user-agent"])

	// Test: Duplicate Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nAccept: a\r\nAccept: b\r\n\r\n",
		numBytesPerRead: 20,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	assert.Equal(t, "a, b", r.Headers["accept"])

	// Test: Leading/Trailing Whitespace
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost \r\nUser-Agent:\tcurl\r\n\r\n",
		numBytesPerRead: 1,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	assert.Equal(t, "localhost", r.Headers["host"])
	assert.Equal(t, "curl", r.Headers["user-agent"])

	// Test: Malformed Variants
	cases := []struct {
		data string
	}{
		{"GET / HTTP/1.1\r\nHost localhost\r\n\r\n"},
		{"GET / HTTP/1.1\r\nHost : val\r\n\r\n"},
		{"GET / HTTP/1.1\r\nH©st: localhost:42069\r\n\r\n"},
	}
	for _, c := range cases {
		for _, chunk := range []int{1, 2, 3, 4, 5, len(c.data)} {
			reader := &chunkReader{data: c.data, numBytesPerRead: chunk}
			_, err := RequestFromReader(reader)
			require.Error(t, err)
		}
	}

	// Test: Multiple Headers in One Read
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl\r\nAccept: a\r\nAccept: b\r\nAccept: c\r\nLanguage: en-US\r\n\r\n",
		numBytesPerRead: len(reader.data),
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	assert.Equal(t, "localhost:42069", r.Headers["host"])
	assert.Equal(t, "curl", r.Headers["user-agent"])
	assert.Equal(t, "a, b, c", r.Headers["accept"])
	assert.Equal(t, "en-US", r.Headers["language"])

	// Test: Missing End of Headers
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\n",
		numBytesPerRead: 3,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Line beginning with ws raises error
	reader = &chunkReader{
		data:            "GET / HTTP/1.1\r\n Foo: bar\r\n\r\n",
		numBytesPerRead: 3,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)
}

func TestParsingBody(t *testing.T) {
	// Test: Standard Body
	reader := &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 13\r\n" +
			"\r\n" +
			"hello world!\n",
		numBytesPerRead: 3,
	}
	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "hello world!\n", string(r.Body))

	// Test: Body shorter than reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 20\r\n" +
			"\r\n" +
			"partial content",
		numBytesPerRead: 3,
	}
	_, err = RequestFromReader(reader)
	require.Error(t, err)

	// Test: Empty Body, 0 reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"Content-Length: 0\r\n" +
			"\r\n",
		numBytesPerRead: 1,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "", string(r.Body))

	// Test: Empty Body, no reported content length
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"\r\n",
		numBytesPerRead: 6,
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	assert.Equal(t, "", string(r.Body))

	// Test: No Content-Length but Body Exists
	reader = &chunkReader{
		data: "POST /submit HTTP/1.1\r\n" +
			"Host: localhost:42069\r\n" +
			"\r\n" +
			"hello world!\n",
		numBytesPerRead: len(reader.data),
	}
	r, err = RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "", string(r.Body))
}
