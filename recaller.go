package recall

import (
	"context"
	"log/slog"
	"strings"
)

var recallKey struct{ Recaller }
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

type Recaller struct {
	context            context.Context
	logger             *slog.Logger
	messageFormat      string
	correlationAttrKey string
}

// New creates a new Recaller initialized with a Context, default logger and default message format.
func New(ctx context.Context) Recaller {
	return Recaller{
		context:       ctx,
		logger:        slog.Default(),
		messageFormat: "[RECALL] %s",
	}
}

// WithContext returns a Recaller with the context.
func (r Recaller) WithContext(ctx context.Context) Recaller {
	r.context = ctx
	return r
}

// WithLogger sets the logger to use for the debug log message.
func (r Recaller) WithLogger(logger *slog.Logger) Recaller {
	r.logger = logger
	return r
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

func (r Recaller) Call(f func(ctx context.Context) error) error {
	// is debug enabled?
	if r.logger.Handler().Enabled(r.context, slog.LevelDebug) {
		// no recall on error needed
		return f(r.context)
	}
	err := f(r.context)
	if err != nil {
		currentLogger := LoggerFromContext(r.context)
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
