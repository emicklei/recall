package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/emicklei/recall"
)

func main() {
	fmt.Println("try each of these:")
	fmt.Println("curl http://localhost:8080/ok")
	fmt.Println("curl http://localhost:8080/fail")
	http.HandleFunc("/{workId}", handleWork)
	http.ListenAndServe(":8080", nil)
}

var requestID atomic.Int32 // for demonstration purposes only

func handleWork(w http.ResponseWriter, r *http.Request) {
	logWithRequestID := slog.Default().With("request-id", requestID.Add(1))
	ctxWithLogger := recall.ContextWithLogger(r.Context(), logWithRequestID)
	recaller := recall.New(ctxWithLogger)

	// wrap doWork in a function to be able to call it again on error
	if err := recaller.Call(func(ctx context.Context) error {

		if err := doWork(ctx, r.PathValue("workId")); err != nil {
			// must use logger from context
			rlog := recall.Slog(ctx)
			rlog.Error("work failed", "err", err)
			return err
		}
		return nil

	}); err != nil {
		// already logged, just return response
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
