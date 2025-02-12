package recall

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

var logKey struct{ slog.Logger }

// ContextWithLogger returns a new context with the logger.
func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, logKey, logger)
}

// LoggerFromContext returns the logger from the context or the default logger if not found.
func LoggerFromContext(ctx context.Context) (l *slog.Logger) {
	v := ctx.Value(logKey)
	if v == nil {
		return slog.Default()
	}
	return v.(*slog.Logger)
}

type captureStrategy int

const (
	RecallOnErrorStrategy captureStrategy = iota
	RecordingStrategy
)

type Recaller struct {
	context         context.Context
	messageFormat   string
	captureStrategy captureStrategy
}

// New creates a new Recaller initialized with a Context, default logger and default message format.
func New(ctx context.Context) Recaller {
	return Recaller{
		context:         ctx,
		messageFormat:   "[RECALL] %s",
		captureStrategy: RecallOnErrorStrategy,
	}
}

// WithMessageFormat sets the message format for the debug log message.
// Must contains a single %s placeholder for the original message.
func (r Recaller) WithMessageFormat(format string) Recaller {
	if !strings.Contains(format, "%s") {
		panic("Recaller message format must contain a single %s placeholder")
	}
	r.messageFormat = format
	return r
}

// WithCaptureStrategy sets the strategy for capturing log messages. Default is RecallOnErrorStrategy.
func (r Recaller) WithCaptureStrategy(strategy captureStrategy) Recaller {
	r.captureStrategy = strategy
	return r
}

// Call calls the function and produces debug log messages when the function returns an error.
// Depending on the capture strategy, the function is called once or twice.
// The default strategy is to call the function a second time when an error is returned.
func (r Recaller) Call(f func(ctx context.Context) error) error {
	if r.captureStrategy == RecordingStrategy {
		return r.captureRecords(f)
	}
	return r.captureStrategyRecallOnError(f)
}

// captureStrategyRecallOnError calls the function and captures debug log messages on the second call
// when the function returns an error.
func (r Recaller) captureStrategyRecallOnError(f func(ctx context.Context) error) error {
	currentLogger := LoggerFromContext(r.context)
	// is debug enabled?
	if currentLogger.Handler().Enabled(r.context, slog.LevelDebug) {
		// no recall on error needed
		return f(r.context)
	}
	err := f(r.context)
	if err != nil {
		handler := debugHandler{currentLogger.Handler(), r.messageFormat}
		debugLogger := slog.New(handler)
		ctx := ContextWithLogger(r.context, debugLogger)
		err = f(ctx)
		if err == nil {
			// this time it worked
			return nil
		}
	}
	return err
}

// captureRecords call the function and records all non-handled log messages.
// If the function returns an error then the recorded messages are replayed.
func (r Recaller) captureRecords(f func(ctx context.Context) error) error {
	def := slog.Default()
	rec := newRecorder(def.Handler())
	log := slog.New(rec)
	ctx := ContextWithLogger(r.context, log)
	err := f(ctx)
	if err != nil {
		rec.recordsDo(func(each slog.Record) {
			attrs := []any{}
			each.Attrs(func(a slog.Attr) bool {
				attrs = append(attrs, a)
				return true
			})
			if each.Level == slog.LevelDebug {
				def.Log(r.context, slog.LevelInfo, fmt.Sprintf(r.messageFormat, each.Message), attrs...)
			} else {
				def.Log(r.context, each.Level, each.Message, attrs...)
			}
		})
	}
	return err
}
