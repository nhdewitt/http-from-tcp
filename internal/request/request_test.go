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
