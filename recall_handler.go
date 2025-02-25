package recall

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"runtime/debug"
	"strings"
)

type logFailedRequestHandler struct {
	next           http.Handler
	messageFormat  string
	handlePanic    bool
	bufferCapacity int
}

// NewRecallHandler uses the RecordingStrategy for capturing logs during HTTP request processing.
// It will write the Debug logs if the request fails (http status >= 400) and details about the HTTP request including the payload.
func NewRecallHandler(next http.Handler) logFailedRequestHandler {
	return logFailedRequestHandler{next: next, messageFormat: "[RECALL] %s", handlePanic: true, bufferCapacity: math.MaxInt}
}

// WithPanicRecovery enables or disables handling panics. Default is true.
// An extra Error log entry is written after recovering from a panic.
func (h logFailedRequestHandler) WithPanicRecovery(enabled bool) logFailedRequestHandler {
	h.handlePanic = enabled
	return h
}

// WithMessageFormat sets the message format for the debug log message.
// Must contains a single %s placeholder for the original message.
func (h logFailedRequestHandler) WithMessageFormat(format string) logFailedRequestHandler {
	if !strings.Contains(format, "%s") {
		panic("Recaller message format must contain a single %s placeholder")
	}
	h.messageFormat = format
	return h
}

// WithRequestBodyCapture sets a limit to the size of the recorded request body for logging on failure.
func (h logFailedRequestHandler) WithRequestBodyCapture(maxBytes int) logFailedRequestHandler {
	h.bufferCapacity = maxBytes
	return h
}

// ServeHTTP implements http.Handler
func (h logFailedRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// record request payload up to buffer capacity
	bodyReader := &limitedBodyRecorder{body: r.Body, limit: h.bufferCapacity, buffer: new(bytes.Buffer)}
	r.Body = bodyReader

	// create context with recording logger
	def := slog.Default()
	rec := newRecorder(def.Handler(), h.messageFormat)
	log := slog.New(rec)
	ctx := ContextWithLogger(r.Context(), log)

	// do not panic
	if h.handlePanic {
		defer func() {
			// recover from first panic
			err := recover()
			if err != nil {
				rec.flush(ctx)
				def.Error(fmt.Sprintf(h.messageFormat, "recovered from panic"),
					"method", r.Method, "url", r.URL, "headers", r.Header,
					"payload", bodyReader.recorded(), "status", http.StatusInternalServerError,
					"err", err, "stack", string(debug.Stack()))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}()
	}

	// serve the request
	responseWriter := &statusCodeRecorder{ResponseWriter: w}
	h.next.ServeHTTP(responseWriter, r.WithContext(ctx))

	// did it fail?
	if responseWriter.statusCode >= http.StatusBadRequest {
		rec.flush(ctx)
		slog.Info(fmt.Sprintf(h.messageFormat, "HTTP request handling failed"), "method", r.Method,
			"url", r.URL, "headers", r.Header, "payload", bodyReader.recorded(), "status", responseWriter.statusCode)
	}
}

type statusCodeRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (h *statusCodeRecorder) WriteHeader(c int) {
	h.statusCode = c
	h.ResponseWriter.WriteHeader(c)
}

type limitedBodyRecorder struct {
	body      io.ReadCloser
	limit     int
	buffer    *bytes.Buffer
	bytesRead int
}

func (l *limitedBodyRecorder) Read(p []byte) (n int, err error) {
	n, err = l.body.Read(p)
	// write to buffer until hit limit
	if size := l.buffer.Len(); size < l.limit {
		max := min(n, l.limit-size)
		l.buffer.Write(p[:max])
	}
	l.bytesRead += n
	return
}
func (l *limitedBodyRecorder) Close() error {
	return l.body.Close()
}
func (l *limitedBodyRecorder) recorded() string {
	s := l.buffer.String()
	if len(l.buffer.Bytes()) < l.bytesRead {
		s = fmt.Sprintf("%s..(%d of %d)", s, l.limit, l.bytesRead)
	}
	return s
}
