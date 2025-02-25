package recall

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecallHandler(t *testing.T) {
	bad := erroringHandler{}
	h := NewRecallHandler(bad)
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", bytes.NewBufferString("test"))
	h.ServeHTTP(rec, req)
}

type erroringHandler struct{ dopanic bool }

func (h erroringHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rlog := Slog(r.Context())

	data, _ := io.ReadAll(r.Body)
	rlog.Debug("processing", "data", string(data))

	if h.dopanic {
		panic("oops")
	}
	w.WriteHeader(500)
}

func TestRecallHandlerPanic(t *testing.T) {
	bad := erroringHandler{dopanic: true}
	h := NewRecallHandler(bad)
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", bytes.NewBufferString("test"))
	h.ServeHTTP(rec, req)
}
