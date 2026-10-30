package middleware

import (
	"compress/gzip"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A bare status with no body (e.g. a throttle Abort(429)) must reach the client as-is.
func TestGzipWriterPassesStatusWithoutBody(t *testing.T) {
	rec := httptest.NewRecorder()
	gw := &gzipResponseWriter{ResponseWriter: rec}

	gw.WriteHeader(nethttp.StatusTooManyRequests)
	require.NoError(t, gw.Close())

	assert.Equal(t, nethttp.StatusTooManyRequests, rec.Code)
	assert.Empty(t, rec.Header().Get("Content-Encoding"), "no gzip header without a gzip body")
	assert.Zero(t, rec.Body.Len())
}

func TestGzipWriterCompressesBodyWithStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	gw := &gzipResponseWriter{ResponseWriter: rec}

	gw.WriteHeader(nethttp.StatusCreated)
	_, err := gw.Write([]byte(`{"ok":true}`))
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	assert.Equal(t, nethttp.StatusCreated, rec.Code)
	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	zr, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)
	body, err := io.ReadAll(zr)
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, string(body))
}
