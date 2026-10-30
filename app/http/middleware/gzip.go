package middleware

import (
	"bufio"
	"compress/gzip"
	"net"
	nethttp "net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
)

// ginInstance 用于从 Goravel 的 Context 获取底层 *gin.Context（仅 gin 驱动时可用）
type ginInstance interface {
	Instance() *gin.Context
}

const gzipEncoding = "gzip"

// gzipResponseWriter 包装 http.ResponseWriter，对 body 做 gzip 压缩
type gzipResponseWriter struct {
	nethttp.ResponseWriter
	code int
	once sync.Once
	gz   *gzip.Writer
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.Header().Set("Content-Encoding", gzipEncoding)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	w.once.Do(func() {
		h := w.ResponseWriter.Header()
		h.Set("Content-Encoding", gzipEncoding)
		h.Del("Content-Length")
		h.Add("Vary", "Accept-Encoding")

		w.gz = gzip.NewWriter(w.ResponseWriter)
		if w.code != 0 {
			w.ResponseWriter.WriteHeader(w.code)
		}
	})
	return w.gz.Write(b)
}

func (w *gzipResponseWriter) Close() error {
	if w.gz != nil {
		return w.gz.Close()
	}
	// No body was written (e.g. Abort with a bare 429): the status is only sent on the first
	// Write, so pass it through now, uncompressed, or the client sees an empty 200.
	if w.code != 0 {
		w.ResponseWriter.Header().Del("Content-Encoding")
		w.ResponseWriter.WriteHeader(w.code)
	}
	return nil
}

// Gzip 仅在本地/开发环境下对响应做 gzip 压缩（线上由 Nginx 等做压缩）
func Gzip() http.Middleware {
	return newMiddleware("gzip", func(ctx http.Context) {
		env := facades.Config().GetString("app.env", "production")
		if env != "local" && env != "development" {
			ctx.Request().Next()
			return
		}

		if strings.EqualFold(ctx.Request().Header("Upgrade", ""), "websocket") {
			ctx.Request().Next()
			return
		}

		// SSE 端点不能做 gzip：gzip 会缓冲全部数据直到流关闭，导致 EventStream 始终为空
		if strings.HasSuffix(ctx.Request().Path(), "/stream") {
			ctx.Request().Next()
			return
		}

		ae := ctx.Request().Header("Accept-Encoding", "")
		if !strings.Contains(strings.ToLower(ae), gzipEncoding) {
			ctx.Request().Next()
			return
		}

		getter, ok := ctx.(ginInstance)
		if !ok {
			ctx.Request().Next()
			return
		}

		ginCtx := getter.Instance()
		origWriter := ginCtx.Writer

		gw := &gzipResponseWriter{ResponseWriter: origWriter}
		defer func() {
			_ = gw.Close()
		}()

		ginCtx.Writer = &ginResponseWriter{ResponseWriter: gw, gzipWriter: gw, size: noWrittenGzip}
		ctx.Request().Next()
	})
}

// ginResponseWriter 实现 gin.ResponseWriter，委托给底层并支持 gzip
type ginResponseWriter struct {
	nethttp.ResponseWriter
	gzipWriter *gzipResponseWriter
	size       int
	status     int
}

const noWrittenGzip = -1
const defaultStatusGzip = nethttp.StatusOK

func (w *ginResponseWriter) WriteHeader(code int) {
	if code > 0 && w.status != code {
		w.status = code
		w.gzipWriter.WriteHeader(code)
	}
}

func (w *ginResponseWriter) Write(b []byte) (int, error) {
	n, err := w.gzipWriter.Write(b)
	w.size += n
	return n, err
}

func (w *ginResponseWriter) WriteString(s string) (int, error) {
	n, err := w.Write([]byte(s))
	return n, err
}

func (w *ginResponseWriter) Status() int {
	if w.status == 0 {
		return defaultStatusGzip
	}
	return w.status
}

func (w *ginResponseWriter) Size() int {
	return w.size
}

func (w *ginResponseWriter) Written() bool {
	return w.size != noWrittenGzip
}

func (w *ginResponseWriter) WriteHeaderNow() {
	// gzip 在首次 Write 时写 header，这里仅保证 status 已传下去
	if w.status != 0 {
		w.gzipWriter.WriteHeader(w.status)
	}
}

func (w *ginResponseWriter) Flush() {
	if w.gzipWriter.gz != nil {
		_ = w.gzipWriter.gz.Flush()
	}
	if f, ok := w.gzipWriter.ResponseWriter.(nethttp.Flusher); ok {
		f.Flush()
	}
}

func (w *ginResponseWriter) Pusher() nethttp.Pusher {
	if p, ok := w.ResponseWriter.(nethttp.Pusher); ok {
		return p
	}
	return nil
}

// CloseNotify 实现 gin.ResponseWriter（已弃用，委托给底层或返回已关闭 channel）
func (w *ginResponseWriter) CloseNotify() <-chan bool {
	if cn, ok := w.gzipWriter.ResponseWriter.(interface{ CloseNotify() <-chan bool }); ok {
		return cn.CloseNotify()
	}
	ch := make(chan bool)
	close(ch)
	return ch
}

// Hijack 实现 gin.ResponseWriter，委托给底层（用于 WebSocket 等）
func (w *ginResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.gzipWriter.ResponseWriter.(nethttp.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, nil
}
