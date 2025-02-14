package recall

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
)

func TestRecorderWithGroup(t *testing.T) {
	def := slog.Default()
	def.WithGroup("g0").Info("ref", "a", "b")
	rec := newRecorder(def.Handler(), "%s")
	log := slog.New(rec).WithGroup("g1")
	log.Debug("test", "a", "b")
	if len(rec.records) != 1 {
		t.Fatal()
	}
	first := rec.records[0]
	attrs := attrsFrom(first)
	if v := attrs[0].Key; v != "g1.a" {
		t.Error("unexpected", v)
	}
	subgrp := log.WithGroup("g2")
	subgrp.Debug("test", "a", "b")
	snd := rec.records[1]
	attrs = attrsFrom(snd)
	if v := attrs[0].Key; v != "g2.a" {
		t.Error("unexpected", v)
	}

}

func TestRecorder(t *testing.T) {
	def := slog.Default()
	rec := newRecorder(def.Handler(), "%s")
	log := slog.New(rec)
	log.Info("welcome", "a", "b")
	log.Debug("to my", "c", "d")
	all := rec.records
	if len(all) != 1 {
		t.Fail()
	}
	if all[0].Message != "to my" {
		t.Fail()
	}
	if all[0].NumAttrs() != 1 {
		t.Errorf("expected 1 attribute, got %d", all[0].NumAttrs())
	}
	rec.flush(context.TODO())
	if len(rec.records) != 0 {
		t.Fail()
	}
}
func TestRecorderWarn(t *testing.T) {
	def := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	rec := newRecorder(def.Handler(), "%s")
	log := slog.New(rec)
	log.Info("welcome", "a", "b")
	log.Debug("to my", "c", "d")
	all := rec.records
	if len(all) != 2 {
		t.Errorf("expected 2 records, got %d", len(all))
	}
	if all[0].Message != "welcome" {
		t.Errorf("expected to my, got %s", all[0].Message)
	}
	if all[0].NumAttrs() != 1 {
		t.Errorf("expected 1 attribute, got %d", all[0].NumAttrs())
	}
}

func TestRecordingWithSubLoggersForAttrs(t *testing.T) {
	def := slog.Default()
	rec := newRecorder(def.Handler(), "%s")
	log := slog.New(rec)
	sub1 := log.With("a", "b")
	sub1.Debug("sub1", "c", "d")
	sub2 := log.With("e", "f")
	sub2.Debug("detail")
	sub3 := sub2.With("g", "h")
	sub3.Debug("end")
	if len(rec.records) != 3 {
		t.Errorf("expected 2 records, got %d", len(rec.records))
		return
	}
	first := rec.records[0]
	if first.Message != "sub1" {
		t.Fail()
	}
	if first.NumAttrs() != 2 {
		t.Errorf("expected 1 attrs, got %d", first.NumAttrs())
	}
	last := rec.records[2]
	if last.Message != "end" {
		t.Fail()
	}
	if last.NumAttrs() != 2 {
		t.Errorf("expected 2 attrs, got %d", last.NumAttrs())
	}
	attrs := attrsFrom(last)
	if v := attrs[0].Key; v != "e" {
		t.Errorf("unexpected:%v", v)
	}
	if v := attrs[1].Key; v != "g" {
		t.Errorf("unexpected:%v", v)
	}
}

func attrsFrom(record slog.Record) (list []slog.Attr) {
	record.Attrs(func(a slog.Attr) bool {
		list = append(list, a)
		return true
	})
	return
}

func TestFlushOnError(t *testing.T) {
	def := slog.Default()
	rec := newRecorder(def.Handler(), "%s")
	log := slog.New(rec)
	log.Debug("test")
	if len(rec.records) != 1 {
		t.Fail()
	}
	log.Error("error")
	if len(rec.records) != 0 {
		t.Fail()
	}
}

type badHandler struct{}

func (b badHandler) Enabled(ctx context.Context, level slog.Level) bool   { return false }
func (b badHandler) Handle(ctx context.Context, record slog.Record) error { return fmt.Errorf("bad") }
func (b badHandler) WithAttrs(attrs []slog.Attr) slog.Handler             { return b }
func (b badHandler) WithGroup(group string) slog.Handler                  { return b }

func TestErrornousDefaultHandler(t *testing.T) {
	def := slog.New(badHandler{})
	rec := newRecorder(def.Handler(), "%s")
	log := slog.New(rec)
	log.Debug("test", "q", 42)
	log.Error("test")
}
