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
	h = h.WithMessageFormat("test %s")
	h = h.WithPanicRecovery(false)
	h = h.WithRequestBodyCapture(100)

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", bytes.NewBufferString("test"))
	h.ServeHTTP(rec, req)
}

func TestRecallHandlerBadMessageFormat(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fatal()
		}
	}()
	NewRecallHandler(nil).WithMessageFormat("")
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

func TestLimitedBodyRecorder(t *testing.T) {
	r := io.NopCloser(bytes.NewReader([]byte("test"))) // 4 bytes to read
	l := &limitedBodyRecorder{body: r, limit: 2, buffer: new(bytes.Buffer)}
	n, _ := l.Read(make([]byte, 3))
	remain, _ := io.ReadAll(l)
	if got, want := n, 3; got != want {
		t.Errorf("got [%[1]v:%[1]T] want [%[2]v:%[2]T]", got, want)
	}
	l.Close()
	if got, want := l.recorded(), "te..(2 of 4)"; got != want {
		t.Errorf("got [%[1]v:%[1]T] want [%[2]v:%[2]T]", got, want)
	}
	if got, want := len(remain), 1; got != want {
		t.Errorf("got [%[1]v:%[1]T] want [%[2]v:%[2]T]", got, want)
	}
}
