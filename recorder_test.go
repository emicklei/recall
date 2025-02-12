package recall

import (
	"log/slog"
	"os"
	"testing"
)

func TestRecorder(t *testing.T) {
	def := slog.Default()
	rec := newRecorder(def.Handler())
	log := slog.New(rec)
	log.Info("welcome", "a", "b")
	log.Debug("to my", "c", "d")
	log.Error("space", "e", "f")
	all := []slog.Record{}
	rec.recordsDo(func(each slog.Record) {
		all = append(all, each)
	})
	if len(all) != 1 {
		t.Fail()
	}
	if all[0].Message != "to my" {
		t.Fail()
	}
	if all[0].NumAttrs() != 1 {
		t.Errorf("expected 1 attribute, got %d", all[0].NumAttrs())
	}
}
func TestRecorderWarn(t *testing.T) {
	def := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	rec := newRecorder(def.Handler())
	log := slog.New(rec)
	log.Info("welcome", "a", "b")
	log.Debug("to my", "c", "d")
	log.Error("space", "e", "f")
	all := []slog.Record{}
	rec.recordsDo(func(each slog.Record) {
		all = append(all, each)
	})
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
