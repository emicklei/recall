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

// alias
var LoggerFromContext = Slog

// Slog returns the slog logger from the context or the default logger if not found.
func Slog(ctx context.Context) (l *slog.Logger) {
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

// WithPanicRecovery enables or disables handling panics. Default is true.
// An extra Error log entry is written after recovering from a panic.
func (r Recaller) WithPanicRecovery(enabled bool) Recaller {
	r.handlePanic = enabled
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
						currentLogger.Error(fmt.Sprintf(r.messageFormat, "recovered from panic"),
							"recall", true, "err", err, "stack", string(debug.Stack()))
						callErr = fmt.Errorf("%v", secondErr)
					}
				}()
				// second time return value could be nil
				callErr = r.callWithDebugLogging(f)
			}
		}()
	}
	err := f(r.context)
	if err != nil {
		// second time return value could be nil
		err = r.callWithDebugLogging(f)
	}
	return err
}

func (r Recaller) callWithDebugLogging(f func(ctx context.Context) error) error {
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
				log.Error(fmt.Sprintf(r.messageFormat, "recovered from panic"),
					"recall", true, "err", err, "stack", string(debug.Stack()))
				callErr = fmt.Errorf("%v", err)
			}
		}()
	}
	err := f(ctx)
	if err != nil {
		rec.flush(ctx)
	}
	return err
}
