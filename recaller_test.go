package recall

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"errors"
)

func TestRecall(t *testing.T) {
	r := New(context.Background())
	err := r.Call(willError)
	if err == nil {
		t.Error("expected error")
	}
	err = r.Call(noError)
	if err != nil {
		t.Error("expected no error")
	}
}

func TestRecallRecords(t *testing.T) {
	rec := new(recording)
	ctx := ContextWithLogger(context.Background(), slog.New(rec))
	r := New(ctx)
	r.Call(willError)
	if len(rec.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(rec.records))
	}

	if !strings.HasSuffix(rec.records[0].Message, "will error") {
		t.Error(rec.records[0].Message)
	}
}

func willError(ctx context.Context) error {
	LoggerFromContext(ctx).Debug("will error")
	return errors.New("error")
}
func noError(ctx context.Context) error {
	LoggerFromContext(ctx).Debug("no error")
	return nil
}

type recording struct {
	records []slog.Record
}

func (r *recording) Enabled(ctx context.Context, level slog.Level) bool {
	return level == slog.LevelInfo
}
func (r *recording) Handle(ctx context.Context, record slog.Record) error {
	r.records = append(r.records, record)
	return nil
}
func (r *recording) WithAttrs(attrs []slog.Attr) slog.Handler {
	return r
}
func (r *recording) WithGroup(group string) slog.Handler {
	return r
}
