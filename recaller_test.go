package recall

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"errors"
)

func TestRecallDisabled(t *testing.T) {
	recallEnabled = false
	t.Cleanup(func() {
		recallEnabled = true
	})
	rec := new(recording)
	def := slog.New(rec)
	r := New(ContextWithLogger(context.Background(), def))
	r.Call(willError)
	if len(rec.records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(rec.records))
	}
}

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
func TestRecallSecondSuccess(t *testing.T) {
	r := New(context.Background())
	err := r.Call(noErrorOnRetry)
	if err != nil {
		t.Error("not expected error")
	}
}
func TestRecallRecording(t *testing.T) {
	r := New(context.Background()).WithCaptureStrategy(RecordingStrategy)
	err := r.Call(willError)
	if err == nil {
		t.Error("expected error")
	}
	err = r.Call(noError)
	if err != nil {
		t.Error("expected no error")
	}
}

func TestRecallMessageFormat(t *testing.T) {
	rec := new(recording)
	ctx := ContextWithLogger(context.Background(), slog.New(rec))
	r := New(ctx).WithMessageFormat("message %s")
	r.Call(willError)
	if len(rec.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(rec.records))
	}
	if rec.records[0].Message != "message will error" {
		t.Error(rec.records[0].Message)
	}
}
func TestRecallBadMessageFormat(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fatal()
		}
	}()
	New(context.Background()).WithMessageFormat("")
}
func TestRecallDefaultHasDebug(t *testing.T) {
	rec := new(recording)
	rec.level = slog.LevelDebug
	def := slog.New(rec)
	r := New(ContextWithLogger(context.Background(), def))
	r.Call(willError)
	if len(rec.records) != 1 {
		t.Fatalf("expected 1 records, got %d", len(rec.records))
	}
}

func TestRecallDefaultHasWarn(t *testing.T) {
	rec := new(recording)
	rec.level = slog.LevelWarn
	def := slog.New(rec)
	r := New(ContextWithLogger(context.Background(), def))
	r.Call(willError)
	if len(rec.records) != 1 {
		t.Fatalf("expected 1 records, got %d", len(rec.records))
	}
}

func TestRecallDefaultHasDebugWhenRecording(t *testing.T) {
	rec := new(recording)
	rec.level = slog.LevelDebug
	def := slog.New(rec)
	r := New(ContextWithLogger(context.Background(), def)).WithCaptureStrategy(RecordingStrategy)
	r.Call(willError)
	if len(rec.records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(rec.records))
	}
}

func TestRecallRecordingWithPanic(t *testing.T) {
	rec := new(recording)
	log := slog.New(rec)
	slog.SetDefault(log)
	ctx := ContextWithLogger(context.Background(), log)
	r := New(ctx).WithCaptureStrategy(RecordingStrategy)
	r.Call(willPanic)
	if len(rec.records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(rec.records))
	}

	if !strings.HasSuffix(rec.records[0].Message, "before panic") {
		t.Error(rec.records[0].Message)
	}
}

func TestRecallRecordingNoPanic(t *testing.T) {
	rec := new(recording)
	log := slog.New(rec)
	slog.SetDefault(log)
	ctx := ContextWithLogger(context.Background(), log)
	r := New(ctx).WithCaptureStrategy(RecordingStrategy).WithPanicRecovery(false)
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
	}()
	r.Call(willPanic)
}

var retries = 0

func noErrorOnRetry(ctx context.Context) error {
	LoggerFromContext(ctx).Debug("will error first time")
	if retries == 0 {
		retries++
		return errors.New("error")
	}
	return nil
}

func willError(ctx context.Context) error {
	LoggerFromContext(ctx).Debug("will error")
	return errors.New("error")
}
func noError(ctx context.Context) error {
	LoggerFromContext(ctx).Debug("no error")
	return nil
}

func TestRecallPanic(t *testing.T) {
	r := New(context.Background())
	err := r.Call(willPanic)
	if err == nil {
		t.Error("expected error")
	}
}

func TestRecallNoPanic(t *testing.T) {
	r := New(context.Background()).WithPanicRecovery(false)
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
	}()
	r.Call(willPanic)
}

func willPanic(ctx context.Context) error {
	LoggerFromContext(ctx).Debug("before panic")
	panic("boom")
}

type recording struct {
	records []slog.Record
	level   slog.Level // info
}

func (r *recording) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= r.level
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
