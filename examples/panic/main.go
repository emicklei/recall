package main

import (
	"context"
	"log/slog"

	"github.com/emicklei/recall"
)

func main() {
	recaller := recall.New(context.Background())

	err := recaller.Call(func(ctx context.Context) error {
		rlog := recall.Slog(ctx)

		rlog.Debug("this will show up on panic")
		doPanic()
		return nil
	})

	slog.Error("bummer", "err", err)
}

func doPanic() { panic("boom") }
