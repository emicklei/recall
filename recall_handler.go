package recall

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
)

type logFailedRequestHandler struct {
	next          http.Handler
	messageFormat string
	handlePanic   bool
}

// NewRecallHandler using the RecordingStrategy for capturing logs during HTTP request processing.
// It will write the Debug logs if the request fails (http status >= 400) and details about the HTTP request including the payload.
func NewRecallHandler(next http.Handler) http.Handler {
	return logFailedRequestHandler{next: next, messageFormat: "[RECALL] %s", handlePanic: true}
}

// ServeHTTP implements http.Handler
func (h logFailedRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// record request payload
	var payload []byte
	if r.Body != nil {
		// store it for logging if processing it fails
		payload, _ = io.ReadAll(r.Body)
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(payload))
	}

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
				log.Error(fmt.Sprintf(h.messageFormat, "recovered from panic"),
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
		slog.Info(fmt.Sprintf(h.messageFormat, "HTTP request handling failed"), "method", r.Method, "url", r.URL, "headers", r.Header, "payload", string(payload), "status", responseWriter.statusCode)
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
