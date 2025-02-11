## interceptor

All functions called on behalf of an intercepted service request need to access the logger from a Context instead of using the `slog` package directly.

    slog := recall.LoggerFromContext(ctx)
    ...
    slog.Debug("this will up on error")