package recall

import (
	"context"
	"log/slog"
)

type recorder struct {
	parent  *recorder
	records []slog.Record
	handler slog.Handler
}

func newRecorder(handler slog.Handler) *recorder {
	return &recorder{
		handler: handler,
	}
}

func (r *recorder) Enabled(ctx context.Context, level slog.Level) bool {
	// we filter in the handle
	return true
}
func (r *recorder) Handle(ctx context.Context, record slog.Record) error {
	// only record those which are not enabled
	if !r.handler.Enabled(ctx, record.Level) {
		r.records = append(r.records, record)
		return nil
	}
	return r.handler.Handle(ctx, record)
}
func (r *recorder) WithAttrs(attrs []slog.Attr) slog.Handler {
	sub := newRecorder(r.handler.WithAttrs(attrs))
	sub.parent = r
	return sub
}
func (r *recorder) WithGroup(group string) slog.Handler {
	sub := newRecorder(r.handler.WithGroup(group))
	sub.parent = r
	return sub
}

func (r *recorder) recordsDo(f func(record slog.Record)) {
	for _, record := range r.records {
		f(record)
	}
	if r.parent != nil {
		r.parent.recordsDo(f)
	}
}
