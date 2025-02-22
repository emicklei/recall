## interceptor

All functions called on behalf of an intercepted service request need to access the logger from a Context instead of using the `slog` package directly.

    rlog := recall.Slog(ctx)
    ...
    rlog.Debug("this will up on error")