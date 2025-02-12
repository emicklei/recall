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
		rlog := recall.LoggerFromContext(ctx)

		rlog.Info("begin")
		rlog.Debug("this will show up on error", "k", "v")
		return errors.New("something went wrong")
	})

	slog.Error("bummer", "err", err)
}

// 2025/02/11 18:55:23 INFO begin
// 2025/02/11 18:55:23 INFO [RECALL] this will show up on error k=v
// 2025/02/11 18:55:23 ERROR bummer err="something went wrong"
