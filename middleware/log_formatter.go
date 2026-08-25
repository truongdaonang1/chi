package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// DefaultLogFormatter is the default log formatter.
var DefaultLogFormatter LogFormatter = &DefaultLogFormatterImpl{
	Logger:  &defaultLogger{out: os.Stdout},
	NoColor: false,
}

// LoggerInterface interface for log output.
type LoggerInterface interface {
	Print(v ...interface{})
}

type defaultLogger struct {
	out io.Writer
	mu  sync.Mutex
}

func (l *defaultLogger) Print(v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprint(l.out, v...)
}

// DefaultLogFormatterImpl formats logs for DefaultLogFormatter.
type DefaultLogFormatterImpl struct {
	Logger  LoggerInterface
	NoColor bool
}

func (l *DefaultLogFormatterImpl) NewLogEntry(r *http.Request) LogEntry {
	entry := &defaultLogEntry{
		DefaultLogFormatterImpl: l,
		request:                 r,
		buf:                     &bytes.Buffer{},
	}

	reqID := GetReqID(r.Context())
	if reqID != "" {
		entry.buf.WriteString(fmt.Sprintf("[%s] ", reqID))
	}

	entry.buf.WriteString(fmt.Sprintf(`"%s `, r.Method))

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	entry.buf.WriteString(fmt.Sprintf("%s://%s%s %s\" ", scheme, r.Host, r.RequestURI, r.Proto))
	entry.buf.WriteString(fmt.Sprintf("from %s - ", r.RemoteAddr))

	return entry
}

type defaultLogEntry struct {
	*DefaultLogFormatterImpl
	request *http.Request
	buf     *bytes.Buffer
}

func (l *defaultLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	rctx := chi.RouteContext(l.request.Context())
	var pattern string
	if rctx != nil {
		pattern = rctx.RoutePattern()
	}

	l.buf.WriteString(fmt.Sprintf("%03d %dB", status, bytes))
	if pattern != "" {
		l.buf.WriteString(fmt.Sprintf(" [%s]", pattern))
	}
	l.buf.WriteString(fmt.Sprintf(" in %s\n", elapsed))

	l.Logger.Print(l.buf.String())
}

func (l *defaultLogEntry) Panic(v interface{}, stack []byte) {
	panicEntry := &defaultLogEntry{
		DefaultLogFormatterImpl: l.DefaultLogFormatterImpl,
		request:                 l.request,
		buf:                     &bytes.Buffer{},
	}
	panicEntry.buf.WriteString(fmt.Sprintf("PANIC: %v\n%s\n", v, string(stack)))
	l.Logger.Print(panicEntry.buf.String())
}
