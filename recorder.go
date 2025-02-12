package recall

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type recorder struct {
	mux           *sync.RWMutex
	handler       slog.Handler
	records       []slog.Record
	messageFormat string
}

type subRecorder struct {
	root  *recorder
	attrs []slog.Attr
	group string
}

func (r subRecorder) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (r subRecorder) Handle(ctx context.Context, record slog.Record) error {
	if r.group == "" {
		record.AddAttrs(r.attrs...)
		return r.root.Handle(ctx, record)
	}
	clone := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(a slog.Attr) bool {
		clone.Add(fmt.Sprintf("%s.%s", r.group, a.Key), a.Value)
		return true
	})
	return r.root.Handle(ctx, clone)
}

func (r subRecorder) WithAttrs(attrs []slog.Attr) slog.Handler {
	return subRecorder{attrs: append(r.attrs, attrs...), root: r.root}
}
func (r subRecorder) WithGroup(group string) slog.Handler {
	return subRecorder{root: r.root, group: group}
}

func newRecorder(handler slog.Handler, format string) *recorder {
	return &recorder{
		handler:       handler,
		mux:           new(sync.RWMutex),
		messageFormat: format,
	}
}

func (r *recorder) Enabled(ctx context.Context, level slog.Level) bool {
	// we filter in the handle
	return true
}

func (r *recorder) Handle(ctx context.Context, record slog.Record) error {
	if record.Level == slog.LevelError {
		r.flush(ctx)
		return r.handler.Handle(ctx, record)
	}
	// only record those which are not enabled
	if !r.handler.Enabled(ctx, record.Level) {
		r.mux.Lock()
		r.records = append(r.records, record)
		r.mux.Unlock()
		return nil
	}
	return r.handler.Handle(ctx, record)
}
func (r *recorder) WithAttrs(attrs []slog.Attr) slog.Handler {
	return subRecorder{attrs: attrs, root: r}
}
func (r *recorder) WithGroup(group string) slog.Handler {
	return subRecorder{root: r, group: group}
}

func (r *recorder) flush(ctx context.Context) {
	r.mux.Lock()
	defer r.mux.Unlock()
	for _, record := range r.records {
		if record.Level == slog.LevelDebug {
			record.Message = fmt.Sprintf(r.messageFormat, record.Message)
			// change level otherwise it will be filtered out
			record.Level = slog.LevelInfo
		}
		r.handler.Handle(ctx, record)
	}
	r.records = []slog.Record{}
}
