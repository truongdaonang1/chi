package middleware

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// WrapResponseWriter wraps http.ResponseWriter to capture status and bytes written.
type WrapResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	Status() int
	BytesWritten() int
	Tee(io.Writer)
	Unwrap() http.ResponseWriter
}

// NewWrapResponseWriter creates a response writer wrapper.
func NewWrapResponseWriter(w http.ResponseWriter, protoMajor int) WrapResponseWriter {
	_, fl := w.(http.Flusher)
	bw := &basicWriter{ResponseWriter: w}
	if fl {
		return &flushWriter{basicWriter: bw}
	}
	return bw
}

type basicWriter struct {
	http.ResponseWriter
	wroteHeader bool
	code        int
	bytes       int
	tee         io.Writer
}

func (b *basicWriter) WriteHeader(code int) {
	if !b.wroteHeader {
		b.code = code
		b.wroteHeader = true
		b.ResponseWriter.WriteHeader(code)
	}
}

func (b *basicWriter) Write(buf []byte) (int, error) {
	b.WriteHeader(http.StatusOK)
	n, err := b.ResponseWriter.Write(buf)
	b.bytes += n
	if b.tee != nil {
		_, _ = b.tee.Write(buf[:n])
	}
	return n, err
}

func (b *basicWriter) Status() int {
	if b.code == 0 {
		return http.StatusOK
	}
	return b.code
}

func (b *basicWriter) BytesWritten() int {
	return b.bytes
}

func (b *basicWriter) Tee(w io.Writer) {
	b.tee = w
}

func (b *basicWriter) Unwrap() http.ResponseWriter {
	return b.ResponseWriter
}

func (b *basicWriter) Flush() {
	if fl, ok := b.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

type flushWriter struct {
	*basicWriter
}

func (f *flushWriter) Flush() {
	if fl, ok := f.basicWriter.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

func (f *flushWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := f.basicWriter.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (f *flushWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := f.basicWriter.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}
