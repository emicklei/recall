package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/emicklei/recall"
)

func main() {
	recaller := recall.New(context.Background()).WithCaptureStrategy(recall.RecordingStrategy)

	err := recaller.Call(func(ctx context.Context) error {
		rlog := recall.Slog(ctx)

		rlog.Info("begin")
		rlog.Debug("this will show up on error", "k", "v")

		err := errors.New("something went wrong")
		rlog.Error("failed", "err", err)
		return err
	})

	slog.Error("bummer", "err", err)
}

// 2025/02/12 15:56:07 INFO begin
// 2025/02/12 15:56:07 INFO [RECALL] this will show up on error k=v
// 2025/02/12 15:56:07 ERROR failed err="something went wrong"
// 2025/02/12 15:56:07 ERROR bummer err="something went wrong"
