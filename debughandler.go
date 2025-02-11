package recall

import (
	"context"
	"fmt"
	"log/slog"
)

// debugHandler is to capture the Handle method of a slog.Handler and change the level of debug messages to info.
type debugHandler struct {
	slog.Handler
	messageFormat string
}

func (d debugHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= slog.LevelDebug
}
func (d debugHandler) Handle(ctx context.Context, rec slog.Record) error {
	// mark the message as a recall
	rec.Message = fmt.Sprintf(d.messageFormat, rec.Message)
	if rec.Level == slog.LevelDebug {
		// change level so that it gets logged
		if d.Handler.Enabled(ctx, slog.LevelInfo) {
			rec.Level = slog.LevelInfo
		} else {
			// if info is not enabled, fallback to warn
			rec.Level = slog.LevelWarn
		}
	}
	return d.Handler.Handle(ctx, rec)
}
