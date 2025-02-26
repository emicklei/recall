package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emicklei/recall"
)

func main() {
	fmt.Println("try each of these:")
	fmt.Println("curl http://localhost:8080/ok")
	fmt.Println(`curl -H "Authorization:secret" http://localhost:8080/fail`)
	http.HandleFunc("/{workId}", handleWork)

	handler := recall.NewRecallHandler(http.DefaultServeMux)
	handler = handler.WithMessageFormat("[DEMO] %s")
	handler = handler.WithPanicRecovery(false)     // override default
	handler = handler.WithRequestBodyCapture(1024) // log only the first 1024 bytes on failure
	handler = handler.WithHeaderFilter(func(in http.Header) (out http.Header) {
		c := in.Clone()
		c.Del("Authorization")
		return c
	})
	http.ListenAndServe(":8080", handler)
}

func handleWork(w http.ResponseWriter, r *http.Request) {
	if err := doWork(r.Context(), r.PathValue("workId")); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func doWork(ctx context.Context, workId string) error {
	// must use rlog from context
	rlog := recall.Slog(ctx)

	rlog.Info("doWork started")

	// this will show up on recall only or once if debug is enabled
	rlog.Debug("doWork debug information", "var", "value")

	return doOtherWork(ctx, workId)
}

func doOtherWork(ctx context.Context, workId string) error {
	// must use rlog from context
	rlog := recall.Slog(ctx)

	// this will show up on recall only or once if debug is enabled
	rlog.Debug("doOtherWork", "workId", workId)

	// simulate something that fails
	if workId == "fail" {
		return fmt.Errorf("invalid wordId: %s", workId)
	}
	return nil
}
