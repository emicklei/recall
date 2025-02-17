package recall

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
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
	handlePanic     bool
}

// New creates a new Recaller initialized with a Context, default logger and default message format.
func New(ctx context.Context) Recaller {
	return Recaller{
		context:         ctx,
		messageFormat:   "[RECALL] %s",
		captureStrategy: RecallOnErrorStrategy,
		handlePanic:     true,
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
func (r Recaller) captureStrategyRecallOnError(f func(ctx context.Context) error) (callErr error) {
	currentLogger := LoggerFromContext(r.context)
	// is debug enabled?
	if currentLogger.Handler().Enabled(r.context, slog.LevelDebug) {
		// no recall on error needed
		return f(r.context)
	}
	if r.handlePanic {
		defer func() {
			// recover from first panic
			err := recover()
			if err != nil {
				defer func() {
					// recover from second panic
					secondErr := recover()
					if secondErr != nil {
						callErr = fmt.Errorf("panic: %v, stack:%s", secondErr, string(debug.Stack()))
					}
				}()
				callErr = r.recoveredCall(f)
			}
		}()
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

func (r Recaller) recoveredCall(f func(ctx context.Context) error) error {
	currentLogger := LoggerFromContext(r.context)
	handler := debugHandler{currentLogger.Handler(), r.messageFormat}
	debugLogger := slog.New(handler)
	ctx := ContextWithLogger(r.context, debugLogger)
	return f(ctx)
}

// captureRecords call the function and records all non-handled log messages.
// If the function returns an error then the recorded messages are replayed.
func (r Recaller) captureRecords(f func(ctx context.Context) error) (callErr error) {
	def := slog.Default()
	rec := newRecorder(def.Handler(), r.messageFormat)
	log := slog.New(rec)
	ctx := ContextWithLogger(r.context, log)
	if r.handlePanic {
		defer func() {
			// recover from first panic
			err := recover()
			if err != nil {
				rec.flush(ctx)
				callErr = fmt.Errorf("panic: %v, stack:%s", err, string(debug.Stack()))
			}
		}()
	}
	err := f(ctx)
	if err != nil {
		rec.flush(ctx)
	}
	return err
}
