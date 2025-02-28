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

type RecallHandler struct {
	next             http.Handler
	messageFormat    string
	handlePanic      bool
	bufferCapacity   int
	headerFilter     func(in http.Header) (out http.Header)
	statusCodeFilter func(statusCode int) bool
}

// NewRecallHandler uses the RecordingStrategy for capturing logs during HTTP request processing.
// It will write the Debug logs if the request fails (http status >= 400) and details about the HTTP request including the payload.
func NewRecallHandler(next http.Handler) RecallHandler {
	return RecallHandler{
		next:           next,
		messageFormat:  "[RECALL] %s",
		handlePanic:    true,
		bufferCapacity: math.MaxInt,
		headerFilter:   nil,
	}
}

// WithPanicRecovery enables or disables handling panics. Default is true.
// An extra Error log entry is written after recovering from a panic.
func (h RecallHandler) WithPanicRecovery(enabled bool) RecallHandler {
	h.handlePanic = enabled
	return h
}

// WithMessageFormat sets the message format for the debug log message.
// Must contains a single %s placeholder for the original message.
func (h RecallHandler) WithMessageFormat(format string) RecallHandler {
	if !strings.Contains(format, "%s") {
		panic("Recaller message format must contain a single %s placeholder")
	}
	h.messageFormat = format
	return h
}

// WithRequestBodyCapture sets a limit to the size of the recorded request body for logging on failure.
func (h RecallHandler) WithRequestBodyCapture(maxBytes int) RecallHandler {
	h.bufferCapacity = maxBytes
	return h
}

// WithHeaderFilter allows you to modify the request headers before producing a log entry.
// This can be used to mask or remove sensitive information such as tokens or cookies.
func (h RecallHandler) WithHeaderFilter(f func(in http.Header) (out http.Header)) RecallHandler {
	h.headerFilter = f
	return h
}

// WithStatusCodeFilter allows you to decide for which HTTP status code you want to produce log entries.
func (h RecallHandler) WithStatusCodeFilter(f func(statusCode int) bool) RecallHandler {
	h.statusCodeFilter = f
	return h
}

// ServeHTTP implements http.Handler
func (h RecallHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
					"method", r.Method, "url", r.URL, "headers", h.filteredHeaders(r.Header),
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
	fail := responseWriter.statusCode >= http.StatusBadRequest
	if h.statusCodeFilter != nil {
		fail = fail && h.statusCodeFilter(responseWriter.statusCode)
	}
	if fail {
		rec.flush(ctx)
		slog.Info(fmt.Sprintf(h.messageFormat, "HTTP request handling failed"), "method", r.Method,
			"url", r.URL, "headers", h.filteredHeaders(r.Header), "payload", bodyReader.recorded(), "status", responseWriter.statusCode)
	}
}

func (h RecallHandler) filteredHeaders(headers http.Header) http.Header {
	if h.headerFilter == nil {
		return headers
	}
	return h.headerFilter(headers)
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
